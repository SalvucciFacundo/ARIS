package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aris/pkg/imgutil"
)

func (r *Runner) handleWorkflow(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: aris workflow <inspect|export> <image_path> [options]")
		return 1
	}

	subcmd := args[0]
	switch subcmd {
	case "help", "-h", "--help":
		fmt.Println("Usage: aris workflow <inspect|export> <image_path> [options]")
		fmt.Println("\nSubcommands:")
		fmt.Println("  inspect <image.png> [--json]       Inspect embedded ComfyUI/ARIS generation metadata")
		fmt.Println("  export <image.png> [-o <path>] [-f] Export raw ComfyUI node graph JSON to file or stdout")
		return 0
	case "inspect":
		return r.handleWorkflowInspect(args[1:])
	case "export":
		return r.handleWorkflowExport(args[1:])
	default:
		fmt.Printf("❌ Unknown workflow subcommand: %q\n", subcmd)
		fmt.Println("Available subcommands: inspect, export")
		return 1
	}
}

func (r *Runner) handleWorkflowInspect(args []string) int {
	if len(args) == 0 {
		fmt.Println("❌ Error: image path is required.")
		fmt.Println("Usage: aris workflow inspect <image_path> [--json]")
		return 1
	}

	inspectFlags := flag.NewFlagSet("workflow inspect", flag.ContinueOnError)
	jsonFlag := inspectFlags.Bool("json", false, "Output metadata in raw JSON format")

	var flagArgs []string
	var imagePath string

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
		} else if imagePath == "" {
			imagePath = arg
		}
	}

	_ = inspectFlags.Parse(flagArgs)

	if imagePath == "" {
		fmt.Println("❌ Error: image path is required.")
		return 1
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		fmt.Printf("❌ Error: image file not found: %s\n", imagePath)
		return 1
	}

	meta, err := imgutil.ExtractPNGMetadataFile(imagePath)
	if err != nil {
		fmt.Printf("❌ Error extracting metadata: %v\n", err)
		return 1
	}

	if *jsonFlag {
		output := make(map[string]any)
		for k, v := range meta {
			var parsed any
			if err := json.Unmarshal([]byte(v), &parsed); err == nil {
				output[k] = parsed
			} else {
				output[k] = v
			}
		}
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Printf("❌ Error encoding JSON: %v\n", err)
			return 1
		}
		fmt.Println(string(jsonBytes))
		return 0
	}

	fmt.Println("🖼️  ARIS Image Workflow & Generation Metadata")
	fmt.Printf("📁 File: %s\n", imagePath)
	fmt.Println("------------------------------------------------------------")

	if len(meta) == 0 {
		fmt.Println("ℹ️  No embedded generation metadata found in this image.")
		return 0
	}

	if params, ok := meta["parameters"]; ok {
		fmt.Printf("📝 Generation Parameters:\n%s\n\n", params)
	}

	if promptStr, ok := meta["prompt"]; ok {
		var promptMap map[string]any
		if err := json.Unmarshal([]byte(promptStr), &promptMap); err == nil {
			fmt.Printf("⚙️  Execution Prompt Graph: %d node(s)\n", len(promptMap))
		}
	}

	if wfStr, ok := meta["workflow"]; ok {
		var wfMap map[string]any
		if err := json.Unmarshal([]byte(wfStr), &wfMap); err == nil {
			if nodes, ok := wfMap["nodes"].([]any); ok {
				fmt.Printf("🎨 ComfyUI Visual Workflow: %d node(s)\n", len(nodes))
			} else {
				fmt.Printf("🎨 ComfyUI Visual Workflow: Embedded (%d bytes)\n", len(wfStr))
			}
		}
	}

	for k := range meta {
		if k != "parameters" && k != "prompt" && k != "workflow" {
			fmt.Printf("🏷️  %s: %s\n", k, meta[k])
		}
	}

	fmt.Println("------------------------------------------------------------")
	fmt.Println("💡 Tip: Use 'aris workflow export <image.png>' to extract ComfyUI node graph JSON.")
	return 0
}

func (r *Runner) handleWorkflowExport(args []string) int {
	if len(args) == 0 {
		fmt.Println("❌ Error: image path is required.")
		fmt.Println("Usage: aris workflow export <image_path> [-o output.json] [--force]")
		return 1
	}

	exportFlags := flag.NewFlagSet("workflow export", flag.ContinueOnError)
	var outputDest string
	exportFlags.StringVar(&outputDest, "o", "", "Destination file path for exported workflow JSON")
	exportFlags.StringVar(&outputDest, "output", "", "Destination file path for exported workflow JSON")
	var forceOverwrite bool
	exportFlags.BoolVar(&forceOverwrite, "f", false, "Overwrite destination file if it exists")
	exportFlags.BoolVar(&forceOverwrite, "force", false, "Overwrite destination file if it exists")

	var flagArgs []string
	var imagePath string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if (arg == "-o" || arg == "--output") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else if imagePath == "" {
			imagePath = arg
		}
	}

	_ = exportFlags.Parse(flagArgs)

	if imagePath == "" {
		fmt.Println("❌ Error: image path is required.")
		return 1
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		fmt.Printf("❌ Error: image file not found: %s\n", imagePath)
		return 1
	}

	meta, err := imgutil.ExtractPNGMetadataFile(imagePath)
	if err != nil {
		fmt.Printf("❌ Error extracting metadata: %v\n", err)
		return 1
	}

	workflowData, hasWf := meta["workflow"]
	if !hasWf || strings.TrimSpace(workflowData) == "" {
		if promptData, hasPrompt := meta["prompt"]; hasPrompt {
			workflowData = promptData
		} else {
			fmt.Printf("❌ Error: No ComfyUI workflow metadata found in %s\n", imagePath)
			return 1
		}
	}

	var parsed any
	if err := json.Unmarshal([]byte(workflowData), &parsed); err != nil {
		fmt.Printf("❌ Error: Embedded workflow metadata is not valid JSON: %v\n", err)
		return 1
	}
	prettyJSON, _ := json.MarshalIndent(parsed, "", "  ")

	if outputDest == "-" {
		fmt.Println(string(prettyJSON))
		return 0
	}

	if outputDest == "" {
		ext := filepath.Ext(imagePath)
		outputDest = strings.TrimSuffix(imagePath, ext) + ".workflow.json"
	}

	if _, err := os.Stat(outputDest); err == nil && !forceOverwrite {
		fmt.Printf("❌ Error: destination file %q already exists. Use -f or --force to overwrite.\n", outputDest)
		return 1
	}

	if err := os.WriteFile(outputDest, prettyJSON, 0644); err != nil {
		fmt.Printf("❌ Error saving workflow file: %v\n", err)
		return 1
	}

	fmt.Printf("✅ ComfyUI workflow successfully exported to: %s\n", outputDest)
	return 0
}
