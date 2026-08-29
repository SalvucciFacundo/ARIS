package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/services"
	"aris/pkg/imgutil"
)

// handleUpscale processes the `aris upscale <image_path> [options]` CLI subcommand.
func (r *Runner) handleUpscale(args []string) int {
	if len(args) < 1 {
		fmt.Println("❌ Error: <image_path> is required.")
		fmt.Println("Usage: aris upscale <image_path> [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  -s, --scale <int>      Scale factor: 2, 4, or 8 (default: 4)")
		fmt.Println("  --restore-faces        Enable face artifact reconstruction (default: false)")
		fmt.Println("  -f, --fidelity <float> Face fidelity weight [0.0 - 1.0] (default: 0.75)")
		fmt.Println("  -b, --backend <str>    Target backend: falai, comfyui (default: user configured)")
		fmt.Println("  -m, --model <str>      Upscaler model name (e.g. fal-ai/esrgan, RealESRGAN_x4plus.pth)")
		fmt.Println("  -o, --output <path>    Explicit output path for upscaled image")
		fmt.Println("  --critic               Run VLM visual critique on output")
		return 1
	}

	upscaleFlags := flag.NewFlagSet("upscale", flag.ContinueOnError)
	scaleFlag := upscaleFlags.Int("scale", 4, "Scale factor: 2, 4, or 8")
	_ = upscaleFlags.Int("s", 4, "Shorthand for scale")
	restoreFacesFlag := upscaleFlags.Bool("restore-faces", false, "Enable face reconstruction")
	fidelityFlag := upscaleFlags.Float64("fidelity", 0.75, "Face fidelity weight [0.0 - 1.0]")
	_ = upscaleFlags.Float64("f", 0.75, "Shorthand for fidelity")
	backendFlag := upscaleFlags.String("backend", "", "Target backend")
	_ = upscaleFlags.String("b", "", "Shorthand for backend")
	modelFlag := upscaleFlags.String("model", "", "Upscaler model name")
	_ = upscaleFlags.String("m", "", "Shorthand for model")
	outputFlag := upscaleFlags.String("output", "", "Explicit output path")
	_ = upscaleFlags.String("o", "", "Shorthand for output")
	criticFlag := upscaleFlags.Bool("critic", false, "Enable VLM critique")

	var positionalArgs []string
	var flagArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	_ = upscaleFlags.Parse(flagArgs)

	if len(positionalArgs) < 1 {
		fmt.Println("❌ Error: <image_path> positional argument is required.")
		fmt.Println("Usage: aris upscale <image_path> [options]")
		return 1
	}

	imagePath := positionalArgs[0]
	if strings.TrimSpace(imagePath) == "" {
		fmt.Println("❌ Error: image path cannot be empty.")
		return 1
	}

	// 1. Validate scale factor
	scale := *scaleFlag
	if scale != 2 && scale != 4 && scale != 8 {
		fmt.Printf("❌ Error: unsupported scale factor %d: must be 2, 4, or 8\n", scale)
		return 1
	}

	// 2. Validate input image
	data, _, err := imgutil.LoadAndValidateImage(imagePath, imgutil.MaxImageSize)
	if err != nil {
		fmt.Printf("❌ Invalid input image %q: %v\n", imagePath, err)
		return 1
	}

	origW, origH, _ := imgutil.GetDimensions(data)

	// 3. Check backend compatibility
	targetBackend := *backendFlag
	if targetBackend == "openai" {
		fmt.Println("❌ Error: OpenAI DALL-E backend does not support standalone image upscaling or face restoration. Please use falai or comfyui.")
		return 1
	}

	// If fidelity is explicitly set without restore-faces flag, enable face restoration
	restoreFaces := *restoreFacesFlag
	fidelity := *fidelityFlag
	for _, a := range flagArgs {
		if strings.HasPrefix(a, "-f") || strings.HasPrefix(a, "--fidelity") {
			restoreFaces = true
			break
		}
	}

	opts := services.GenerateOptions{
		InputImage:    imagePath,
		Mode:          domain.ModeUpscale,
		ScaleFactor:   scale,
		RestoreFaces:  restoreFaces,
		FaceFidelity:  fidelity,
		UpscalerModel: *modelFlag,
		Backend:       targetBackend,
		EnableCritic:  *criticFlag,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	fmt.Printf("🔍 ARIS Super-Resolution & Face Restoration Pipeline\n")
	fmt.Printf("🖼️ Input Image:     %s (%dx%d)\n", imagePath, origW, origH)
	fmt.Printf("⚡ Scale Factor:    %dx -> Target: %dx%d\n", scale, origW*scale, origH*scale)
	if restoreFaces {
		fmt.Printf("👤 Face Restore:    Enabled (Fidelity: %.2f)\n", fidelity)
	} else {
		fmt.Printf("👤 Face Restore:    Disabled\n")
	}
	if *modelFlag != "" {
		fmt.Printf("🤖 Model:           %s\n", *modelFlag)
	}

	start := time.Now()
	spec, result, err := r.agent.Generate(ctx, "upscale image", opts)
	if err != nil {
		fmt.Printf("❌ Upscaling failed: %v\n", err)
		return 1
	}

	outputPath := result.LocalPath
	if *outputFlag != "" {
		customOut := *outputFlag
		_ = os.MkdirAll(filepath.Dir(customOut), 0755)
		if err := copyFile(result.LocalPath, customOut); err == nil {
			outputPath = customOut
		}
	}

	fmt.Printf("\n✨ Image Upscaled Successfully in %v!\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("📐 Final Multiplier: %dx (%dx%d)\n", spec.ScaleFactor, origW*scale, origH*scale)
	fmt.Printf("🔌 Backend:          %s\n", spec.Backend)
	if result.Metadata != nil {
		if score, ok := result.Metadata["critic_score"]; ok {
			fmt.Printf("👁️ VLM Score:        %.2f\n", score)
		}
	}
	fmt.Printf("💾 Saved to:         %s\n", outputPath)
	return 0
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
