package cli

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/services"
)

// handleBatch processes the `aris batch "<prompt>" [options]` CLI subcommand.
func (r *Runner) handleBatch(args []string) int {
	if len(args) == 0 {
		fmt.Println("❌ Error: prompt is required.")
		fmt.Println("Usage: aris batch \"<prompt>\" [options]")
		r.printBatchHelp()
		return 1
	}

	batchFlags := flag.NewFlagSet("batch", flag.ContinueOnError)

	var (
		countVal       int
		sweepVal       string
		seedVal        int64
		matrixVal      bool
		benchVal       bool
		backendsVal    string
		concurrencyVal int
		outputDirVal   string
		maxJobsVal     int
		forceVal       bool
		dryRunVal      bool
		evalVal        bool
		ratioVal       string
		modelVal       string
		negVal         string
	)

	batchFlags.IntVar(&countVal, "count", 1, "Number of image variants to generate")
	batchFlags.IntVar(&countVal, "c", 1, "Shorthand for count")
	batchFlags.StringVar(&sweepVal, "seed-sweep", "", "Deterministic seed sweep range (<start>-<end>)")
	batchFlags.StringVar(&sweepVal, "s", "", "Shorthand for seed-sweep")
	batchFlags.Int64Var(&seedVal, "seed", 0, "Base seed for count generation")
	batchFlags.BoolVar(&matrixVal, "matrix", false, "Enable prompt matrix Cartesian expansion")
	batchFlags.BoolVar(&matrixVal, "m", false, "Shorthand for matrix")
	batchFlags.BoolVar(&benchVal, "benchmark", false, "Enable multi-backend benchmarking mode")
	batchFlags.BoolVar(&benchVal, "b", false, "Shorthand for benchmark")
	batchFlags.StringVar(&backendsVal, "backends", "", "Comma-separated list of image backends")
	batchFlags.IntVar(&concurrencyVal, "concurrency", 2, "Number of concurrent workers")
	batchFlags.IntVar(&concurrencyVal, "j", 2, "Shorthand for concurrency")
	batchFlags.StringVar(&outputDirVal, "output-dir", "", "Custom directory path for batch outputs")
	batchFlags.StringVar(&outputDirVal, "o", "", "Shorthand for output-dir")
	batchFlags.IntVar(&maxJobsVal, "max-matrix-jobs", 100, "Safety upper bound on combinatorial matrix jobs")
	batchFlags.BoolVar(&forceVal, "force", false, "Bypass matrix job limit safety check")
	batchFlags.BoolVar(&dryRunVal, "dry-run", false, "Preview planned jobs without generating images")
	batchFlags.BoolVar(&evalVal, "eval", false, "Enable VLM visual critic evaluation on output images")
	batchFlags.StringVar(&ratioVal, "ratio", "1:1", "Aspect ratio (1:1, 16:9, 9:16, 4:3, 3:4, 21:9)")
	batchFlags.StringVar(&ratioVal, "r", "1:1", "Shorthand for ratio")
	batchFlags.StringVar(&modelVal, "model", "flux", "Generation model")
	batchFlags.StringVar(&negVal, "negative", "", "Negative prompt")
	batchFlags.StringVar(&negVal, "n", "", "Shorthand for negative")

	var positionalArgs []string
	var flagArgs []string

	hasExplicitCount := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			if arg == "--count" || arg == "-c" || strings.HasPrefix(arg, "--count=") || strings.HasPrefix(arg, "-c=") {
				hasExplicitCount = true
			}
			flagArgs = append(flagArgs, arg)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if err := batchFlags.Parse(flagArgs); err != nil {
		return 1
	}

	rawPrompt := strings.TrimSpace(strings.Join(positionalArgs, " "))
	if rawPrompt == "" {
		fmt.Println("❌ Error: prompt cannot be empty.")
		return 1
	}

	// 1. Validate --count
	if countVal <= 0 {
		fmt.Println("❌ Error: --count must be an integer >= 1.")
		return 1
	}

	// 2. Validate mutual exclusion of --count and --seed-sweep
	if sweepVal != "" && hasExplicitCount && countVal > 1 {
		fmt.Println("❌ Error: --count and --seed-sweep are mutually exclusive.")
		return 1
	}

	// 3. Resolve backends
	reg := r.agent.Registry()
	var selectedBackends []string
	if backendsVal != "" {
		parts := strings.Split(backendsVal, ",")
		for _, p := range parts {
			name := strings.TrimSpace(p)
			if name == "" {
				continue
			}
			if _, err := reg.Get(name); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
				return 1
			}
			selectedBackends = append(selectedBackends, name)
		}
	} else if benchVal {
		selectedBackends = reg.List()
	} else {
		def := reg.GetDefault()
		if def != nil {
			selectedBackends = []string{def.Name()}
		} else {
			selectedBackends = []string{"pollinations"}
		}
	}

	if len(selectedBackends) == 0 {
		fmt.Println("❌ Error: no valid image backends selected.")
		return 1
	}

	// 4. Matrix prompt expansion
	var prompts []string
	isMatrix := matrixVal || (strings.Contains(rawPrompt, "[") && strings.Contains(rawPrompt, "]") && strings.Contains(rawPrompt, "|"))
	if isMatrix {
		matrixEngine := services.NewMatrixEngine(maxJobsVal, forceVal)
		expanded, err := matrixEngine.Expand(rawPrompt)
		if err != nil {
			fmt.Printf("❌ Matrix expansion error: %v\n", err)
			return 1
		}
		prompts = expanded
	} else {
		prompts = []string{rawPrompt}
	}

	// 5. Seeds resolution
	var seeds []int64
	if sweepVal != "" {
		parsedSeeds, err := services.ParseSeedSweep(sweepVal)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			return 1
		}
		seeds = parsedSeeds
	} else {
		count := countVal
		seeds = make([]int64, count)
		if seedVal > 0 {
			for i := 0; i < count; i++ {
				seeds[i] = seedVal + int64(i)
			}
		} else {
			rSource := rand.New(rand.NewSource(time.Now().UnixNano()))
			for i := 0; i < count; i++ {
				seeds[i] = rSource.Int63n(9000000) + 1000000
			}
		}
	}

	cfg := services.BatchConfig{
		Count:          countVal,
		SeedSweep:      sweepVal,
		BaseSeed:       seedVal,
		Matrix:         isMatrix,
		Benchmark:      benchVal,
		Backends:       selectedBackends,
		Concurrency:    concurrencyVal,
		OutputDir:      outputDirVal,
		MaxMatrixJobs:  maxJobsVal,
		Force:          forceVal,
		DryRun:         dryRunVal,
		Eval:           evalVal,
		Model:          modelVal,
		AspectRatio:    domain.ParseAspectRatio(ratioVal),
		NegativePrompt: negVal,
	}

	plan, err := services.BuildPlan(cfg, prompts, seeds, selectedBackends)
	if err != nil {
		fmt.Printf("❌ Failed to construct batch plan: %v\n", err)
		return 1
	}

	if outputDirVal != "" {
		plan.OutputDir = outputDirVal
	} else {
		plan.OutputDir = filepath.Join("outputs", plan.BatchID)
	}

	// 6. Handle --dry-run
	if dryRunVal {
		fmt.Printf("📋 ARIS Batch Plan (Dry Run): %d Jobs Planned\n", len(plan.Jobs))
		fmt.Printf("   Batch ID:    %s\n", plan.BatchID)
		fmt.Printf("   Prompts:     %d variant(s)\n", len(prompts))
		fmt.Printf("   Seeds:       %d seed(s)\n", len(seeds))
		fmt.Printf("   Backends:    %s\n", strings.Join(selectedBackends, ", "))
		fmt.Printf("   Concurrency: %d\n\n", concurrencyVal)
		fmt.Println("Planned Job Matrix:")
		fmt.Println("----------------------------------------------------------------------------------------")
		fmt.Printf("| %-5s | %-12s | %-10s | %-50s |\n", "Index", "Backend", "Seed", "Prompt")
		fmt.Println("----------------------------------------------------------------------------------------")
		for _, job := range plan.Jobs {
			pSnippet := job.Prompt
			if len(pSnippet) > 47 {
				pSnippet = pSnippet[:47] + "..."
			}
			fmt.Printf("| %-5d | %-12s | %-10d | %-50s |\n", job.Index, job.Backend, job.Seed, pSnippet)
		}
		fmt.Println("----------------------------------------------------------------------------------------")
		fmt.Printf("\n✨ Dry run complete. 0 images rendered.\n")
		return 0
	}

	// 7. Execute Batch
	fmt.Printf("🚀 Starting ARIS Batch Generation: %d jobs across %d backend(s)\n",
		len(plan.Jobs), len(selectedBackends))
	fmt.Printf("📁 Output directory: %s\n", plan.OutputDir)
	fmt.Printf("⚡ Concurrency: %d workers\n\n", cfg.Concurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Printf("\n🛑 Interrupted! Cancelling worker pool and flushing partial contact sheet...\n")
		cancel()
	}()

	runner := services.NewBatchRunner(reg, nil)
	runner.SetProgressCallback(func(result services.BatchJobResult, completed, total int) {
		pct := float64(completed) / float64(total) * 100.0
		icon := "✅"
		if result.Status != "SUCCESS" {
			icon = "❌"
		}
		fmt.Printf("[%d/%d] %5.1f%% %s Job %d [%s | Seed: %d] (%.2fs)\n",
			completed, total, pct, icon, result.Job.Index, result.Job.Backend, result.Job.Seed, result.Duration.Seconds())
	})

	startTime := time.Now()
	summary, err := runner.Execute(ctx, plan)
	if err != nil && summary == nil {
		fmt.Printf("❌ Batch execution failed: %v\n", err)
		return 1
	}

	// 8. Export Contact Sheet and Manifests
	exporter := services.NewContactSheetExporter(plan.OutputDir)
	if err := exporter.Export(summary); err != nil {
		fmt.Printf("⚠️ Failed to export contact sheet: %v\n", err)
	}

	fmt.Println("\n========================================================")
	fmt.Printf("✨ Batch Generation Complete in %v!\n", time.Since(startTime).Round(time.Millisecond))
	fmt.Printf("📊 Summary: %d Total | ✅ %d Succeeded | ❌ %d Failed\n",
		summary.TotalJobs, summary.SuccessCount, summary.FailedCount)
	fmt.Printf("🌐 Visual Contact Sheet: %s\n", filepath.Join(plan.OutputDir, "index.html"))
	fmt.Printf("📝 Markdown Report:      %s\n", filepath.Join(plan.OutputDir, "summary.md"))
	fmt.Printf("💾 Machine Manifest:     %s\n", filepath.Join(plan.OutputDir, "batch_meta.json"))
	fmt.Println("========================================================")

	return 0
}

func (r *Runner) printBatchHelp() {
	fmt.Println(`
Options for 'batch':
  -c, --count <int>          Number of image variants to generate (default: 1)
  -s, --seed-sweep <start-end> Deterministic sequential seed range (e.g. 100-105)
  --seed <int64>             Base seed for deterministic count generation
  -m, --matrix               Enable prompt matrix Cartesian expansion [opt1|opt2]
  -b, --benchmark            Benchmark prompt across all registered image backends
  --backends <list>          Comma-separated list of backends to execute
  -j, --concurrency <int>    Number of concurrent generation workers (default: 2)
  -o, --output-dir <path>    Custom output directory for batch bundle
  --max-matrix-jobs <int>    Safety upper limit on matrix jobs (default: 100)
  --force                    Bypass matrix upper limit safety check
  --dry-run                  Preview planned job combinations without rendering
  --eval                     Run vision critic VLM evaluation on outputs
  -r, --ratio <ratio>        Aspect ratio: 1:1, 16:9, 9:16, 4:3, 3:4, 21:9
  -m, --model <name>         Model identifier (default: flux)
  -n, --negative <prompt>    Negative prompt keywords

Examples:
  aris batch "a [cyberpunk|steampunk] cat in [Tokyo|London]" --seed-sweep 1-3
  aris batch "neon sports car" --count 4 --backends comfyui,falai --concurrency 4
  aris batch "portrait of an astronaut" --benchmark --dry-run
  aris batch "alien flora [blue|purple]" --eval -o ./outputs/alien_batch`)
}
