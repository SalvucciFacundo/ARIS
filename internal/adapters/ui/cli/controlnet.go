package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"aris/internal/core/domain"
	"aris/pkg/imgutil"
)

// ParseControlNetFlags parses CLI flags for ControlNet conditioning.
// Accepts "<type>:<strength>:<path>", "<type>:<path>", or "<type>".
func ParseControlNetFlags(flags []string) ([]domain.ControlNetConfig, error) {
	var configs []domain.ControlNetConfig
	for _, f := range flags {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		parts := strings.Split(f, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			tokens := strings.Split(part, ":")
			if len(tokens) == 1 {
				cnType := strings.TrimSpace(tokens[0])
				configs = append(configs, domain.ControlNetConfig{
					Type:     cnType,
					Strength: 1.0,
				})
			} else if len(tokens) == 2 {
				cnType := strings.TrimSpace(tokens[0])
				path := strings.TrimSpace(tokens[1])
				configs = append(configs, domain.ControlNetConfig{
					Type:           cnType,
					Strength:       1.0,
					ReferenceImage: path,
					InputPath:      path,
				})
			} else if len(tokens) == 3 {
				cnType := strings.TrimSpace(tokens[0])
				strength, err := strconv.ParseFloat(strings.TrimSpace(tokens[1]), 64)
				if err != nil {
					return nil, fmt.Errorf("invalid controlnet strength in %q: %w", part, err)
				}
				path := strings.TrimSpace(tokens[2])
				configs = append(configs, domain.ControlNetConfig{
					Type:           cnType,
					Strength:       strength,
					ReferenceImage: path,
					InputPath:      path,
				})
			} else {
				return nil, fmt.Errorf("invalid controlnet flag format %q (expected <type>:<strength>:<path> or <type>:<path>)", part)
			}
		}
	}
	return configs, nil
}

// handleControlNet processes `aris controlnet [types|preproc|help]` subcommands.
func (r *Runner) handleControlNet(args []string) int {
	if len(args) == 0 {
		args = []string{"types"}
	}

	subcmd := strings.ToLower(args[0])
	switch subcmd {
	case "types", "list":
		return r.listControlNetTypes()
	case "preproc", "preprocess":
		return r.handlePreprocess(args[1:])
	case "help", "-h", "--help":
		r.printControlNetHelp()
		return 0
	default:
		fmt.Printf("Unknown controlnet command: %s\n", subcmd)
		r.printControlNetHelp()
		return 1
	}
}

func (r *Runner) listControlNetTypes() int {
	fmt.Println("🧩 ARIS Supported ControlNet Structural Conditioning Types:")
	fmt.Println()
	fmt.Println("  1. canny     - High-contrast edge detection map (built-in pure-Go preprocessor)")
	fmt.Println("  2. depth     - Monocular 3D depth estimation map")
	fmt.Println("  3. openpose  - Human anatomical keypoints & skeletal pose detection")
	fmt.Println("  4. lineart   - Artistic outline and line-drawing structural conditioning")
	fmt.Println("  5. scribble  - Hand-drawn sketch and contour guidance")
	fmt.Println()
	fmt.Println("Usage example:")
	fmt.Println("  aris gen \"cyberpunk street\" --controlnet \"canny:0.85:pose.png\" --backend comfyui")
	return 0
}

func (r *Runner) handlePreprocess(args []string) int {
	if len(args) < 2 {
		fmt.Println("❌ Error: <type> and <input_image> are required.")
		fmt.Println("Usage: aris controlnet preproc <type> <input_image> [--output <output_image>]")
		return 1
	}

	cnType := strings.ToLower(strings.TrimSpace(args[0]))
	inputPath := args[1]

	var outputFlag string
	preprocFlags := flag.NewFlagSet("preproc", flag.ContinueOnError)
	preprocFlags.StringVar(&outputFlag, "output", "", "Output path for preprocessed image")
	preprocFlags.StringVar(&outputFlag, "o", "", "Shorthand for output")

	if len(args) > 2 {
		_ = preprocFlags.Parse(args[2:])
	}

	if outputFlag == "" {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputFlag = fmt.Sprintf("%s_%s_edges.png", base, cnType)
	}

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("❌ Input image file does not exist: %s\n", inputPath)
		return 1
	}

	switch cnType {
	case "canny":
		fmt.Printf("⚙️ Running pure-Go Canny edge detection on %s...\n", inputPath)
		if err := imgutil.PreprocessCannyFile(inputPath, outputFlag, 100.0, 200.0); err != nil {
			fmt.Printf("❌ Preprocessing failed: %v\n", err)
			return 1
		}
		fmt.Printf("✅ Saved Canny edge map to: %s\n", outputFlag)
		return 0
	default:
		fmt.Printf("⚠️ Preprocessing for type %q is not locally required (pass-through directly to backend).\n", cnType)
		return 0
	}
}

func (r *Runner) printControlNetHelp() {
	fmt.Println(`Usage: aris controlnet [command]

Commands:
  types, list       List all supported ControlNet conditioning types
  preproc <type> <input> [--output <out>] Run local preprocessor (e.g. Canny edge detector)
  help              Show this help message

Examples:
  aris controlnet types
  aris controlnet preproc canny input.png --output edges.png
  aris gen "portrait" --controlnet "canny:0.8:edges.png"`)
}
