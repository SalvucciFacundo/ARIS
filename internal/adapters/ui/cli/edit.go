package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/services"
	"aris/pkg/imgutil"
)

// handleEdit processes the `aris edit <image_path> "<prompt>" [options]` CLI subcommand.
func (r *Runner) handleEdit(args []string) int {
	if len(args) < 2 {
		fmt.Println("❌ Error: <image_path> and \"<prompt>\" are required.")
		fmt.Println("Usage: aris edit <image_path> \"<prompt>\" [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  --mask <path>       Inpainting mask image path")
		fmt.Println("  -s, --strength      Denoise strength [0.0 - 1.0] (default: 0.70)")
		fmt.Println("  -b, --backend       Backend provider: falai, comfyui, openai, pollinations")
		fmt.Println("  -m, --model         Model identifier")
		fmt.Println("  -r, --ratio         Aspect ratio override (default: 1:1)")
		fmt.Println("  -n, --negative      Negative prompt keywords")
		fmt.Println("  --critic            Run VLM visual critique on output")
		fmt.Println("  --auto-heal         Auto-correct and re-roll on low critique score")
		return 1
	}

	editFlags := flag.NewFlagSet("edit", flag.ContinueOnError)
	maskFlag := editFlags.String("mask", "", "Inpainting mask image path")
	strengthFlag := editFlags.Float64("strength", 0.70, "Denoise strength [0.0 - 1.0]")
	_ = editFlags.Float64("s", 0.70, "Shorthand for strength")
	backendFlag := editFlags.String("backend", "", "Backend provider")
	_ = editFlags.String("b", "", "Shorthand for backend")
	modelFlag := editFlags.String("model", "", "Model identifier")
	_ = editFlags.String("m", "", "Shorthand for model")
	ratioFlag := editFlags.String("ratio", "1:1", "Aspect ratio")
	_ = editFlags.String("r", "1:1", "Shorthand for ratio")
	negFlag := editFlags.String("negative", "", "Negative prompt")
	_ = editFlags.String("n", "", "Shorthand for negative")
	seedFlag := editFlags.Int64("seed", 0, "Seed value")
	criticFlag := editFlags.Bool("critic", false, "Enable VLM critique")
	autoHealFlag := editFlags.Bool("auto-heal", false, "Enable self-healing loop")
	loraFlag := editFlags.String("lora", "", "LoRA model stacking (<name>:<scale>)")
	cnetFlag := editFlags.String("controlnet", "", "ControlNet conditioning (<type>:<scale>:<path>)")

	var positionalArgs []string
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
			positionalArgs = append(positionalArgs, arg)
		}
	}

	_ = editFlags.Parse(flagArgs)
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

	if len(positionalArgs) < 2 {
		fmt.Println("❌ Error: both <image_path> and \"<prompt>\" positional arguments are required.")
		return 1
	}

	imagePath := positionalArgs[0]
	rawPrompt := strings.Join(positionalArgs[1:], " ")

	if strings.TrimSpace(imagePath) == "" || strings.TrimSpace(rawPrompt) == "" {
		fmt.Println("❌ Error: image path and prompt cannot be empty.")
		return 1
	}

	// 1. Validate base image input
	if _, _, err := imgutil.LoadAndValidateImage(imagePath, imgutil.MaxImageSize); err != nil {
		fmt.Printf("❌ Invalid input image %q: %v\n", imagePath, err)
		return 1
	}

	// 2. Validate mask input if provided
	maskPath := strings.TrimSpace(*maskFlag)
	if maskPath != "" {
		if err := imgutil.ValidateMask(imagePath, maskPath, imgutil.MaxImageSize); err != nil {
			fmt.Printf("❌ Mask validation failed: %v\n", err)
			return 1
		}
	}

	mode := domain.ModeImg2Img
	if maskPath != "" {
		mode = domain.ModeInpaint
	}

	opts := services.GenerateOptions{
		AspectRatio:     domain.ParseAspectRatio(*ratioFlag),
		Model:           *modelFlag,
		Backend:         *backendFlag,
		Seed:            *seedFlag,
		NegativePrompt:  *negFlag,
		InputImage:      imagePath,
		MaskImage:       maskPath,
		DenoiseStrength: *strengthFlag,
		Mode:            mode,
		LoRAs:           loraConfigs,
		ControlNets:     cnetConfigs,
		EnableCritic:    *criticFlag,
		AutoHeal:        *autoHealFlag,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	fmt.Printf("🎨 ARIS Visual Reference Pipeline [%s]\n", mode)
	fmt.Printf("🖼️ Base Image: %s\n", imagePath)
	if maskPath != "" {
		fmt.Printf("🎭 Mask Image: %s\n", maskPath)
	}
	fmt.Printf("🎚️ Denoise Strength: %.2f\n", opts.DenoiseStrength)
	fmt.Printf("🧠 Prompt: %q\n", rawPrompt)

	start := time.Now()
	spec, result, err := r.agent.Generate(ctx, rawPrompt, opts)
	if err != nil {
		fmt.Printf("❌ Visual editing failed: %v\n", err)
		return 1
	}

	fmt.Printf("\n✨ Image Edited Successfully in %v!\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("📐 Dimensions: %dx%d (Ratio: %s)\n", spec.Width, spec.Height, spec.AspectRatio)
	fmt.Printf("🔌 Backend:    %s (%s)\n", spec.Backend, spec.Model)
	if result.Metadata != nil {
		if score, ok := result.Metadata["critic_score"]; ok {
			fmt.Printf("👁️ VLM Score:  %.2f\n", score)
		}
	}
	fmt.Printf("💾 Output:     %s\n", result.LocalPath)
	return 0
}
