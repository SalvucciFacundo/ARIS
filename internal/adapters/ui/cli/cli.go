package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"aris/internal/adapters/db"
	"aris/internal/adapters/gateway"
	"aris/internal/adapters/gateway/discord"
	"aris/internal/adapters/gateway/telegram"
	"aris/internal/adapters/image"
	"aris/internal/adapters/llm"
	"aris/internal/adapters/ui/tui"
	"aris/internal/adapters/vision"
	"aris/internal/config"
	"aris/internal/core/domain"
	"aris/internal/core/ports"
	"aris/internal/core/services"
)

// Version of ARIS
const Version = "v1.0.0-dev"

// Runner manages CLI execution.
type Runner struct {
	cfg   *config.Config
	agent *services.AgentService
	db    *db.SQLiteDB
}

// NewRunner sets up database, LLM provider, backend, and agent service.
func NewRunner() (*Runner, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	sqlDB, err := db.NewDefaultSQLiteDB()
	if err != nil {
		return nil, fmt.Errorf("init sqlite db: %w", err)
	}

	kg := db.NewKnowledgeGraph(sqlDB.DB())
	history := db.NewHistoryStore(sqlDB.DB())

	var llmProvider ports.LLMProvider
	if cfg.LLM.Provider != "passthrough" && cfg.LLM.APIKey != "" {
		llmProvider = llm.NewOpenAIClient(cfg.LLM.Provider, cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
	} else {
		llmProvider = llm.NewPassthroughProvider()
	}

	// Setup Image Backend Registry with multi-backend suite
	reg := image.NewRegistry()
	_ = reg.Register(image.NewPollinationsBackend(image.WithOutputDir(cfg.Image.OutputDir)))
	_ = reg.Register(image.NewComfyUIBackend(cfg.Image.ComfyUIHost, cfg.Image.OutputDir, nil))
	_ = reg.Register(image.NewFalAIBackend(cfg.Image.FalKey, cfg.Image.OutputDir, nil))
	_ = reg.Register(image.NewReplicateBackend(cfg.Image.ReplicateToken, cfg.Image.OutputDir, nil))
	_ = reg.Register(image.NewOpenAIBackend(cfg.Image.OpenAIKey, "", cfg.Image.OutputDir, nil))
	_ = reg.Register(image.NewHuggingFaceBackend(cfg.Image.HFToken, cfg.Image.OutputDir, nil))

	if cfg.Image.DefaultBackend != "" {
		_ = reg.SetDefault(cfg.Image.DefaultBackend)
	}

	var visionCritic ports.VisionCritic
	if cfg.Critic.Enabled || os.Getenv("ARIS_ENABLE_CRITIC") == "true" {
		visionCritic = vision.NewVisionClient(
			cfg.Critic.Provider,
			cfg.Critic.APIKey,
			cfg.Critic.BaseURL,
			cfg.Critic.Model,
			nil,
		)
	}

	agent := services.NewAgentService(llmProvider, reg, kg, history, visionCritic)
	subStore := db.NewSubagentStore(sqlDB.DB())
	subMgr := services.NewSubagentManager(subStore, llmProvider, reg, visionCritic, kg)
	agent.SetSubagents(subMgr)

	return &Runner{
		cfg:   cfg,
		agent: agent,
		db:    sqlDB,
	}, nil
}

// Close releases resources.
func (r *Runner) Close() {
	if r.db != nil {
		_ = r.db.Close()
	}
}

// Execute parses args and runs the corresponding command.
func (r *Runner) Execute(args []string) int {
	if len(args) < 2 {
		r.printHelp()
		return 0
	}

	switch args[1] {
	case "chat", "tui", "interactive":
		return r.handleTUI()
	case "gateway", "gw", "server":
		return r.handleGateway(args[2:])
	case "serve", "ui", "web":
		return r.handleServe(args[2:])
	case "gui", "desktop":
		return r.handleGUI(args[2:])
	case "gen", "generate":
		return r.handleGen(args[2:])
	case "edit":
		return r.handleEdit(args[2:])
	case "upscale", "superres":
		return r.handleUpscale(args[2:])
	case "batch":
		return r.handleBatch(args[2:])
	case "lora":
		return r.handleLoRA(args[2:])
	case "controlnet", "cnet":
		return r.handleControlNet(args[2:])
	case "subagents", "subagent", "sub":
		return r.handleSubagents(args[2:])
	case "backends", "backend":
		return r.handleBackends(args[2:])
	case "memory", "mem":
		return r.handleMemory(args[2:])
	case "history", "hist":
		return r.handleHistory(args[2:])
	case "version", "-v", "--version":
		fmt.Printf("ARIS (Autonomous Reasoner for Image System) %s\n", Version)
		return 0
	case "help", "-h", "--help":
		r.printHelp()
		return 0
	default:
		// If user types: aris "a cat on the moon"
		return r.handleGen(args[1:])
	}
}

func (r *Runner) printHelp() {
	fmt.Printf(`ARIS — Autonomous Reasoner for Image System (%s)

Usage:
  aris <command> [options]
  aris gen "<prompt>" [options]
  aris batch "<prompt>" [options]
  aris edit <image_path> "<prompt>" [options]
  aris upscale <image_path> [options]
  aris gen "@director <prompt>"
  aris subagents [list|show|run]

Commands:
  serve, ui           Launch ARIS Web Interface & REST/SSE Server
  gui, desktop        Launch ARIS Desktop GUI (local window or remote VPS client)
  gateway, gw         Launch remote messaging gateway (Telegram & Discord)
  chat, tui           Launch interactive Cyberpunk TUI (split-screen chat & controls)
  gen, generate       Synthesize image from natural language prompt (or route via @name)
  batch               Batch generation, prompt matrix expansion & A/B benchmarking
  edit                Transform or inpaint reference images (img2img / inpaint)
  upscale, superres   Super-resolution scaling (2x, 4x, 8x) & facial reconstruction
  lora                Inspect and manage local LoRA weight models
  controlnet, cnet    Inspect ControlNet types and run local edge preprocessors
  subagents, sub      Inspect and run specialized visual subagents
  backends, backend   List and inspect available local & cloud image backends
  memory, mem         Manage 3-scope Knowledge Graph (list, add, search)
  history, hist       View past generations log
  version             Show version info
  help                Show this help message

Options for 'gen' and 'edit':
  -b, --backend       Image backend: pollinations, comfyui, falai, replicate, openai, huggingface
  -m, --model         Model name (e.g. flux, flux-realism, dall-e-3, sd-3.5)
  -r, --ratio         Aspect ratio: 1:1, 16:9, 9:16, 4:3, 3:4, 21:9 (default: 1:1)
  -s, --strength      Denoise strength [0.0 - 1.0] for edit (default: 0.70)
  --lora <name:scale> LoRA weight model to apply (can be repeated or comma-separated)
  --controlnet <spec> ControlNet structural conditioning: <type>:<scale>:<image_path>
  --mask <path>       Mask image for inpainting (with 'edit')
  -n, --negative      Negative prompt keywords
  --critic            Run VLM visual critique on generated output
  --auto-heal         Automatically retry if critique score is below threshold

Options for 'upscale':
  -s, --scale         Scale multiplier: 2, 4, or 8 (default: 4)
  --restore-faces     Enable face artifact reconstruction (default: false)
  -f, --fidelity      Face fidelity preservation weight [0.0 - 1.0] (default: 0.75)
  -b, --backend       Target backend: falai, comfyui
  -m, --model         Upscaler model name (e.g. fal-ai/esrgan, RealESRGAN_x4plus.pth)
  -o, --output        Explicit output file destination

Examples:
  aris gen "a cyberpunk cat in neo tokyo" --ratio 16:9 --backend pollinations
  aris edit input.png "cyberpunk neon overhaul" --strength 0.65 --backend falai
  aris edit portrait.png "remove glasses" --mask mask.png --backend comfyui
  aris upscale photo.png --scale 4 --restore-faces --fidelity 0.80 -b falai
  aris upscale lowres.png -s 2 -o /tmp/hd_image.png
  aris gen "@director a neon cyberpunk alley in neo tokyo"
  aris gen "@promptsmith: hyperrealistic portrait of a space explorer"
  aris subagents list
  aris subagents show inpainter
  aris subagents show restyler
  aris subagents show upscaler
`, Version)
}

func (r *Runner) handleGateway(args []string) int {
	var adapters []gateway.GatewayAdapter

	queue := gateway.NewJobQueue(r.cfg.Gateway.Concurrency, r.cfg.Gateway.MaxQueue)
	bridge := gateway.NewEngineBridge(r.agent, r.cfg, queue)

	if r.cfg.Gateway.Telegram.Enabled && r.cfg.Gateway.Telegram.BotToken != "" {
		tgAdapter := telegram.NewAdapter(r.cfg.Gateway.Telegram, bridge, queue)
		adapters = append(adapters, tgAdapter)
	}

	if r.cfg.Gateway.Discord.Enabled && r.cfg.Gateway.Discord.BotToken != "" {
		dcAdapter := discord.NewAdapter(r.cfg.Gateway.Discord, bridge, queue)
		adapters = append(adapters, dcAdapter)
	}

	if len(adapters) == 0 {
		fmt.Println("❌ Error: No gateway adapters are enabled.")
		fmt.Println("Please set TELEGRAM_BOT_TOKEN or DISCORD_BOT_TOKEN (or configure ~/.aris/config.yaml).")
		return 1
	}

	mux := gateway.NewMultiplexer(adapters, queue)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Printf("🌐 Starting ARIS Gateway Multiplexer (Concurrency: %d, MaxQueue: %d)...\n",
		r.cfg.Gateway.Concurrency, r.cfg.Gateway.MaxQueue)

	if err := mux.Start(ctx); err != nil {
		fmt.Printf("❌ Failed to start gateway multiplexer: %v\n", err)
		return 1
	}

	// Trap termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	sig := <-sigCh
	fmt.Printf("\n🛑 Received signal (%v), initiating graceful shutdown...\n", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := mux.Stop(shutdownCtx); err != nil {
		fmt.Printf("⚠️ Error during graceful shutdown: %v\n", err)
		return 1
	}

	fmt.Println("✨ ARIS Gateway shutdown complete.")
	return 0
}

func (r *Runner) handleBackends(args []string) int {
	reg := r.agent.Registry()
	backends := reg.List()
	def := reg.GetDefault()
	defName := ""
	if def != nil {
		defName = def.Name()
	}

	fmt.Printf("🔌 ARIS Registered Image Backends (%d available):\n\n", len(backends))
	for i, name := range backends {
		b, _ := reg.Get(name)
		marker := "  "
		if name == defName {
			marker = "⭐ [DEFAULT] "
		}
		fmt.Printf("%d. %s%s\n", i+1, marker, name)
		if len(b.SupportsModels()) > 0 {
			fmt.Printf("   Supported Models: %s\n", strings.Join(b.SupportsModels(), ", "))
		}
		fmt.Println()
	}
	return 0
}

func (r *Runner) handleTUI() int {
	if err := tui.Run(r.agent); err != nil {
		fmt.Printf("❌ TUI error: %v\n", err)
		return 1
	}
	return 0
}

func (r *Runner) handleGen(args []string) int {
	if len(args) == 0 {
		fmt.Println("❌ Error: prompt is required.")
		fmt.Println("Usage: aris gen \"<your prompt>\" [options]")
		return 1
	}

	genFlags := flag.NewFlagSet("gen", flag.ContinueOnError)
	ratioFlag := genFlags.String("ratio", "1:1", "Aspect ratio (1:1, 16:9, 9:16, 4:3, 3:4, 21:9)")
	_ = genFlags.String("r", "1:1", "Shorthand for ratio")
	modelFlag := genFlags.String("model", "flux", "Generation model")
	_ = genFlags.String("m", "flux", "Shorthand for model")
	seedFlag := genFlags.Int64("seed", 0, "Seed value")
	_ = genFlags.Int64("s", 0, "Shorthand for seed")
	negFlag := genFlags.String("negative", "", "Negative prompt")
	_ = genFlags.String("n", "", "Shorthand for negative")
	backendFlag := genFlags.String("backend", "pollinations", "Backend provider")
	_ = genFlags.String("b", "pollinations", "Shorthand for backend")
	criticFlag := genFlags.Bool("critic", false, "Enable VLM vision critique")
	autoHealFlag := genFlags.Bool("auto-heal", false, "Enable automated self-healing re-roll")
	loraFlag := genFlags.String("lora", "", "LoRA model stacking (<name>:<scale>)")
	cnetFlag := genFlags.String("controlnet", "", "ControlNet conditioning (<type>:<scale>:<path>)")

	// Parse flags while leaving raw prompt intact
	var promptParts []string
	var flagArgs []string
	var loraRawFlags []string
	var cnetRawFlags []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--lora" && i+1 < len(args) {
			loraRawFlags = append(loraRawFlags, args[i+1])
			i++
		} else if strings.HasPrefix(arg, "--lora=") {
			loraRawFlags = append(loraRawFlags, strings.TrimPrefix(arg, "--lora="))
		} else if arg == "--controlnet" && i+1 < len(args) {
			cnetRawFlags = append(cnetRawFlags, args[i+1])
			i++
		} else if strings.HasPrefix(arg, "--controlnet=") {
			cnetRawFlags = append(cnetRawFlags, strings.TrimPrefix(arg, "--controlnet="))
		} else if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			promptParts = append(promptParts, arg)
		}
	}

	_ = genFlags.Parse(flagArgs)
	if *loraFlag != "" {
		loraRawFlags = append(loraRawFlags, *loraFlag)
	}
	if *cnetFlag != "" {
		cnetRawFlags = append(cnetRawFlags, *cnetFlag)
	}

	loraConfigs, err := ParseLoRAFlags(loraRawFlags)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return 1
	}

	cnetConfigs, err := ParseControlNetFlags(cnetRawFlags)
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return 1
	}

	rawPrompt := strings.Join(promptParts, " ")
	if strings.TrimSpace(rawPrompt) == "" {
		fmt.Println("❌ Error: prompt cannot be empty.")
		return 1
	}

	// Detect direct subagent routing (@director, @promptsmith, @critic, @curator, @enhancer)
	if subName, cleanPrompt, isSub := services.ParseSubagentRoute(rawPrompt); isSub {
		if cleanPrompt == "" {
			fmt.Printf("❌ Error: input prompt required for subagent @%s\n", subName)
			return 1
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fmt.Printf("🤖 Routing to Subagent @%s: %q\n", subName, cleanPrompt)
		resp, err := r.agent.ExecuteSubagent(ctx, subName, cleanPrompt)
		if err != nil {
			fmt.Printf("❌ Subagent execution failed: %v\n", err)
			return 1
		}
		fmt.Printf("\n💬 Response from @%s:\n%s\n", subName, resp)
		return 0
	}

	opts := services.GenerateOptions{
		AspectRatio:    domain.ParseAspectRatio(*ratioFlag),
		Model:          *modelFlag,
		Backend:        *backendFlag,
		Seed:           *seedFlag,
		NegativePrompt: *negFlag,
		LoRAs:          loraConfigs,
		ControlNets:    cnetConfigs,
		EnableCritic:   *criticFlag,
		AutoHeal:       *autoHealFlag,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	fmt.Printf("🎨 ARIS Reasoning over: %q\n", rawPrompt)
	fmt.Printf("⏳ Consulting Knowledge Graph & dispatching to %s (%s)...\n", opts.Backend, opts.Model)

	start := time.Now()
	spec, result, err := r.agent.Generate(ctx, rawPrompt, opts)
	if err != nil {
		fmt.Printf("❌ Generation failed: %v\n", err)
		return 1
	}

	fmt.Printf("\n✨ Image Generated Successfully in %v!\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("📐 Dimensions: %dx%d (Ratio: %s)\n", spec.Width, spec.Height, spec.AspectRatio)
	fmt.Printf("🌱 Seed:       %d\n", spec.Seed)
	fmt.Printf("🧠 Prompt:     %s\n", spec.EnhancedPrompt)
	if spec.NegativePrompt != "" {
		fmt.Printf("🚫 Negative:   %s\n", spec.NegativePrompt)
	}
	if result.Metadata != nil {
		if score, ok := result.Metadata["critic_score"]; ok {
			fmt.Printf("👁️ VLM Score:  %.2f\n", score)
		}
		if notes, ok := result.Metadata["critic_notes"]; ok {
			fmt.Printf("📝 VLM Notes:  %v\n", notes)
		}
		if healed, ok := result.Metadata["self_healed"]; ok && healed == true {
			fmt.Println("🩹 Self-Healed: Automated prompt correction & re-roll applied!")
		}
	}
	fmt.Printf("💾 Saved to:   %s\n", result.LocalPath)
	return 0
}

func (r *Runner) handleMemory(args []string) int {
	if len(args) == 0 {
		args = []string{"list"}
	}

	subcmd := args[0]
	ctx := context.Background()

	switch subcmd {
	case "list", "ls":
		scope := ""
		if len(args) > 1 {
			scope = args[1]
		}
		facts, err := r.agent.SearchMemory(ctx, "", domain.MemoryScope(scope), 50)
		if err != nil {
			fmt.Printf("❌ Failed to list memory: %v\n", err)
			return 1
		}
		if len(facts) == 0 {
			fmt.Println("🧠 Knowledge Graph is currently empty.")
			return 0
		}
		fmt.Printf("🧠 Knowledge Graph Facts (%d items):\n\n", len(facts))
		for _, f := range facts {
			fmt.Printf("[%s] %s -> %s: %s\n", f.Scope, f.Topic, f.Concept, f.Fact)
		}
		return 0

	case "search", "find":
		if len(args) < 2 {
			fmt.Println("Usage: aris memory search \"<keyword>\"")
			return 1
		}
		query := strings.Join(args[1:], " ")
		facts, err := r.agent.SearchMemory(ctx, query, "", 20)
		if err != nil {
			fmt.Printf("❌ Search failed: %v\n", err)
			return 1
		}
		if len(facts) == 0 {
			fmt.Printf("No facts found matching %q.\n", query)
			return 0
		}
		fmt.Printf("🔍 Matching Facts for %q (%d results):\n\n", query, len(facts))
		for _, f := range facts {
			fmt.Printf("[%s] %s -> %s: %s\n", f.Scope, f.Topic, f.Concept, f.Fact)
		}
		return 0

	case "add":
		addFlags := flag.NewFlagSet("memory add", flag.ExitOnError)
		topic := addFlags.String("topic", "pref:general", "Topic name")
		concept := addFlags.String("concept", "general", "Concept key")
		fact := addFlags.String("fact", "", "Fact description")
		scope := addFlags.String("scope", "style", "Scope: user, style, project")

		_ = addFlags.Parse(args[1:])

		if *fact == "" {
			fmt.Println("❌ Error: --fact is required.")
			return 1
		}

		id, err := r.agent.LearnFact(ctx, *topic, *concept, *fact, domain.MemoryScope(*scope), []string{*topic, *concept})
		if err != nil {
			fmt.Printf("❌ Failed to add fact: %v\n", err)
			return 1
		}
		fmt.Printf("✅ Saved fact [%s] with ID: %s\n", *topic, id)
		return 0

	default:
		fmt.Printf("Unknown memory command: %s\n", subcmd)
		fmt.Println("Available: list, search, add")
		return 1
	}
}

func (r *Runner) handleHistory(args []string) int {
	ctx := context.Background()
	records, err := r.agent.GetHistory(ctx, 20, 0)
	if err != nil {
		fmt.Printf("❌ Failed to get history: %v\n", err)
		return 1
	}

	if len(records) == 0 {
		fmt.Println("📜 No past generations found.")
		return 0
	}

	fmt.Printf("📜 Past Generations History (%d items):\n\n", len(records))
	for i, rec := range records {
		fmt.Printf("%d. [%s] %s (%dx%d, seed %d) -> %s\n",
			i+1, rec.CreatedAt.Format("2006-01-02 15:04"), rec.PromptRaw, rec.Width, rec.Height, rec.Seed, rec.ImagePath)
	}
	return 0
}

func (r *Runner) handleSubagents(args []string) int {
	if len(args) == 0 {
		args = []string{"list"}
	}

	subcmd := args[0]
	ctx := context.Background()
	sm := r.agent.Subagents()
	if sm == nil {
		fmt.Println("❌ Error: Subagent manager not initialized.")
		return 1
	}

	switch subcmd {
	case "list", "ls":
		subagents, err := sm.ListSubagents(ctx)
		if err != nil {
			fmt.Printf("❌ Failed to list subagents: %v\n", err)
			return 1
		}
		fmt.Printf("🤖 ARIS Specialized Visual Subagents (%d active):\n\n", len(subagents))
		for i, sub := range subagents {
			fmt.Printf("%d. @%s — %s (%s)\n", i+1, sub.Name, sub.DisplayName, sub.Role)
			fmt.Printf("   🌡️ Temp: %.2f | 🛠️ Tools: %s\n", sub.Temperature, strings.Join(sub.AllowedTools, ", "))
			fmt.Printf("   📝 %s\n\n", sub.Description)
		}
		return 0

	case "show", "info", "get":
		if len(args) < 2 {
			fmt.Println("Usage: aris subagents show <name>")
			return 1
		}
		name := args[1]
		sub, err := sm.GetSubagent(ctx, name)
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return 1
		}
		fmt.Printf("🤖 Subagent @%s (%s)\n", sub.Name, sub.DisplayName)
		fmt.Printf("Role:         %s\n", sub.Role)
		fmt.Printf("Temperature:  %.2f\n", sub.Temperature)
		fmt.Printf("Tools:        %s\n", strings.Join(sub.AllowedTools, ", "))
		fmt.Printf("Description:  %s\n", sub.Description)
		fmt.Printf("Personality:  %s\n", sub.Personality)
		fmt.Printf("\n📜 System Prompt:\n%s\n", sub.SystemPrompt)
		return 0

	case "run", "exec":
		if len(args) < 3 {
			fmt.Println("Usage: aris subagents run <name> \"<prompt>\"")
			return 1
		}
		name := args[1]
		prompt := strings.Join(args[2:], " ")
		fmt.Printf("🤖 Running subagent @%s with prompt: %q\n", name, prompt)
		resp, err := sm.ExecuteDirect(ctx, name, prompt)
		if err != nil {
			fmt.Printf("❌ Subagent execution failed: %v\n", err)
			return 1
		}
		fmt.Printf("\n💬 Response from @%s:\n%s\n", name, resp)
		return 0

	default:
		fmt.Printf("Unknown subagent command: %s\n", subcmd)
		fmt.Println("Usage: aris subagents [list|show <name>|run <name> \"<prompt>\"]")
		return 1
	}
}
