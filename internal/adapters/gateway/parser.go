package gateway

import (
	"strconv"
	"strings"

	"aris/internal/core/domain"
	"aris/internal/core/services"
)

// CommandType identifies the intent of an incoming gateway message.
type CommandType string

const (
	CmdGen       CommandType = "gen"
	CmdPipeline  CommandType = "pipeline"
	CmdSubagent  CommandType = "subagent"
	CmdSubagents CommandType = "subagents"
	CmdBackends  CommandType = "backends"
	CmdMemory    CommandType = "memory"
	CmdStatus    CommandType = "status"
	CmdHelp      CommandType = "help"
	CmdUnknown   CommandType = "unknown"
)

// ParsedOptions holds parsed execution options for generation, pipeline, or subagent tasks.
type ParsedOptions struct {
	GenOpts        services.GenerateOptions
	PipeOpts       services.PipelineOptions
	SubagentName   string
	SendAsDocument bool
}

// ParsedMessage encapsulates the parsed command, clean prompt, and flags.
type ParsedMessage struct {
	Type        CommandType
	CleanPrompt string
	Options     ParsedOptions
}

// ParseMessage parses raw incoming text from Telegram or Discord.
func ParseMessage(raw string) ParsedMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ParsedMessage{Type: CmdHelp}
	}

	// 1. Check for slash commands
	if strings.HasPrefix(trimmed, "/") {
		fields := strings.Fields(trimmed)
		cmdToken := strings.ToLower(fields[0])
		// Remove bot mention suffix if present (e.g. /gen@aris_bot)
		if idx := strings.Index(cmdToken, "@"); idx != -1 {
			cmdToken = cmdToken[:idx]
		}

		argsStr := strings.TrimSpace(trimmed[len(fields[0]):])

		switch cmdToken {
		case "/gen", "/generate":
			prompt, opts := parseGenArgs(argsStr)
			return ParsedMessage{
				Type:        CmdGen,
				CleanPrompt: prompt,
				Options:     opts,
			}
		case "/pipeline", "/pipe":
			prompt, opts := parseGenArgs(argsStr)
			opts.PipeOpts = services.PipelineOptions{
				AspectRatio:    opts.GenOpts.AspectRatio,
				Model:          opts.GenOpts.Model,
				Backend:        opts.GenOpts.Backend,
				Seed:           opts.GenOpts.Seed,
				NegativePrompt: opts.GenOpts.NegativePrompt,
				EnableCritic:   opts.GenOpts.EnableCritic,
			}
			return ParsedMessage{
				Type:        CmdPipeline,
				CleanPrompt: prompt,
				Options:     opts,
			}
		case "/subagents", "/subagent", "/sub":
			return ParsedMessage{
				Type:        CmdSubagents,
				CleanPrompt: argsStr,
			}
		case "/backends", "/backend":
			return ParsedMessage{
				Type:        CmdBackends,
				CleanPrompt: argsStr,
			}
		case "/memory", "/mem":
			return ParsedMessage{
				Type:        CmdMemory,
				CleanPrompt: argsStr,
			}
		case "/status":
			return ParsedMessage{
				Type:        CmdStatus,
				CleanPrompt: argsStr,
			}
		case "/help", "/start":
			return ParsedMessage{
				Type:        CmdHelp,
				CleanPrompt: argsStr,
			}
		default:
			// Unrecognized slash command, treat as help or unknown
			return ParsedMessage{
				Type:        CmdHelp,
				CleanPrompt: argsStr,
			}
		}
	}

	// 2. Check for @subagent prefix routing (e.g. @director, @promptsmith)
	if subName, cleanPrompt, isSub := services.ParseSubagentRoute(trimmed); isSub {
		return ParsedMessage{
			Type:        CmdSubagent,
			CleanPrompt: cleanPrompt,
			Options: ParsedOptions{
				SubagentName: subName,
			},
		}
	}

	// 3. Fallback: Treat normal text as a direct /gen prompt
	prompt, opts := parseGenArgs(trimmed)
	return ParsedMessage{
		Type:        CmdGen,
		CleanPrompt: prompt,
		Options:     opts,
	}
}

func parseGenArgs(raw string) (string, ParsedOptions) {
	tokens := strings.Fields(raw)
	var promptTokens []string

	opts := ParsedOptions{
		GenOpts: services.GenerateOptions{
			AspectRatio: domain.RatioSquare,
		},
	}

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch {
		case t == "--ratio" || t == "-r":
			if i+1 < len(tokens) {
				opts.GenOpts.AspectRatio = domain.ParseAspectRatio(tokens[i+1])
				i++
			}
		case strings.HasPrefix(t, "--ratio="):
			val := strings.TrimPrefix(t, "--ratio=")
			opts.GenOpts.AspectRatio = domain.ParseAspectRatio(val)
		case t == "--model" || t == "-m":
			if i+1 < len(tokens) {
				opts.GenOpts.Model = tokens[i+1]
				i++
			}
		case strings.HasPrefix(t, "--model="):
			opts.GenOpts.Model = strings.TrimPrefix(t, "--model=")
		case t == "--backend" || t == "-b":
			if i+1 < len(tokens) {
				opts.GenOpts.Backend = tokens[i+1]
				i++
			}
		case strings.HasPrefix(t, "--backend="):
			opts.GenOpts.Backend = strings.TrimPrefix(t, "--backend=")
		case t == "--seed" || t == "-s":
			if i+1 < len(tokens) {
				if s, err := strconv.ParseInt(tokens[i+1], 10, 64); err == nil {
					opts.GenOpts.Seed = s
				}
				i++
			}
		case strings.HasPrefix(t, "--seed="):
			if s, err := strconv.ParseInt(strings.TrimPrefix(t, "--seed="), 10, 64); err == nil {
				opts.GenOpts.Seed = s
			}
		case t == "--negative" || t == "-n":
			if i+1 < len(tokens) {
				opts.GenOpts.NegativePrompt = tokens[i+1]
				i++
			}
		case strings.HasPrefix(t, "--negative="):
			opts.GenOpts.NegativePrompt = strings.TrimPrefix(t, "--negative=")
		case t == "--critic":
			opts.GenOpts.EnableCritic = true
		case t == "--auto-heal":
			opts.GenOpts.AutoHeal = true
		case t == "--doc" || t == "--document":
			opts.SendAsDocument = true
		case t == "--image" || t == "-i":
			if i+1 < len(tokens) {
				opts.GenOpts.InputImage = tokens[i+1]
				i++
			}
		default:
			promptTokens = append(promptTokens, t)
		}
	}

	cleanPrompt := strings.Join(promptTokens, " ")
	return cleanPrompt, opts
}
