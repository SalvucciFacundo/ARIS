package prompt

import (
	"regexp"
	"strconv"
	"strings"

	"aris/internal/core/domain"
)

var (
	// loraRegex matches <lora:name:scale> or <lora:name> tags case-insensitively.
	loraRegex = regexp.MustCompile(`(?i)<lora:([^:>]+)(?::([0-9.]+))?>`)
	// spaceRegex collapses multiple consecutive whitespace characters.
	spaceRegex = regexp.MustCompile(`\s+`)
	// commaSpaceRegex cleans up loose spaces before commas or double commas left after tag removal.
	commaSpaceRegex = regexp.MustCompile(`\s*,\s*`)
)

// ExtractLoRAs parses inline <lora:name:scale> tags from the raw prompt,
// extracts the LoRA configurations, and returns the sanitized clean prompt.
func ExtractLoRAs(rawPrompt string) (string, []domain.LoRAConfig) {
	var loras []domain.LoRAConfig

	matches := loraRegex.FindAllStringSubmatch(rawPrompt, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(match[1])
			scale := 1.0
			if len(match) > 2 && match[2] != "" {
				if parsedScale, err := strconv.ParseFloat(match[2], 64); err == nil {
					scale = parsedScale
				}
			}
			loras = append(loras, domain.LoRAConfig{
				Name:  name,
				Scale: scale,
			})
		}
	}

	// Remove all matched tags from raw prompt
	cleaned := loraRegex.ReplaceAllString(rawPrompt, "")

	// Clean up formatting
	cleaned = cleanWhitespace(cleaned)

	return cleaned, loras
}

// MergeLoRAs merges base LoRAs with additional (e.g. CLI flag) LoRAs.
// If a LoRA with the same name exists, the latter overrides the former's scale.
func MergeLoRAs(base []domain.LoRAConfig, additional []domain.LoRAConfig) []domain.LoRAConfig {
	result := make([]domain.LoRAConfig, 0, len(base)+len(additional))
	indexMap := make(map[string]int)

	for _, l := range base {
		indexMap[strings.ToLower(l.Name)] = len(result)
		result = append(result, l)
	}

	for _, l := range additional {
		key := strings.ToLower(l.Name)
		if idx, exists := indexMap[key]; exists {
			result[idx] = l
		} else {
			indexMap[key] = len(result)
			result = append(result, l)
		}
	}

	return result
}

// cleanWhitespace collapses redundant whitespaces and fixes punctuation spacing.
func cleanWhitespace(s string) string {
	s = spaceRegex.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	// Replace double commas or stray commas like ", ," or ",,"
	for strings.Contains(s, ",,") || strings.Contains(s, ", ,") {
		s = strings.ReplaceAll(s, ", ,", ",")
		s = strings.ReplaceAll(s, ",,", ",")
	}
	s = strings.Trim(s, " ,")
	// Normalize ", "
	parts := strings.Split(s, ",")
	var cleanedParts []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cleanedParts = append(cleanedParts, trimmed)
		}
	}
	return strings.Join(cleanedParts, ", ")
}
