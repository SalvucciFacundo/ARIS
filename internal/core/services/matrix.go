package services

import (
	"fmt"
	"strings"
)

// MatrixEngine expands bracketed combinatorial options in prompts into Cartesian product permutations.
type MatrixEngine struct {
	MaxJobs int
	Force   bool
}

// NewMatrixEngine initializes a new MatrixEngine.
func NewMatrixEngine(maxJobs int, force bool) *MatrixEngine {
	if maxJobs <= 0 {
		maxJobs = 100
	}
	return &MatrixEngine{
		MaxJobs: maxJobs,
		Force:   force,
	}
}

type segmentType int

const (
	segmentLiteral segmentType = iota
	segmentOptions
)

type promptSegment struct {
	kind    segmentType
	literal string
	options []string
}

// Expand parses the prompt for bracketed options [opt1|opt2] and generates all combinatorial permutations.
func (m *MatrixEngine) Expand(prompt string) ([]string, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	segments, err := parsePromptSegments(prompt)
	if err != nil {
		return nil, err
	}

	// Calculate total combinations to enforce MaxJobs limit before generation
	totalCombinations := 1
	for _, seg := range segments {
		if seg.kind == segmentOptions {
			if len(seg.options) > 0 {
				totalCombinations *= len(seg.options)
			}
		}
	}

	if m.MaxJobs > 0 && totalCombinations > m.MaxJobs && !m.Force {
		return nil, fmt.Errorf("planned matrix size (%d) exceeds the maximum allowed limit (%d)", totalCombinations, m.MaxJobs)
	}

	// Cartesian expansion
	results := []string{""}
	for _, seg := range segments {
		if seg.kind == segmentLiteral {
			for i := range results {
				results[i] += seg.literal
			}
		} else if seg.kind == segmentOptions {
			var newResults []string
			for _, prefix := range results {
				for _, opt := range seg.options {
					newResults = append(newResults, prefix+opt)
				}
			}
			results = newResults
		}
	}

	return results, nil
}

func parsePromptSegments(prompt string) ([]promptSegment, error) {
	var segments []promptSegment
	var currentLit strings.Builder

	i := 0
	n := len(prompt)

	for i < n {
		// Check for escaped brackets: \[ or \]
		if prompt[i] == '\\' && i+1 < n && (prompt[i+1] == '[' || prompt[i+1] == ']') {
			currentLit.WriteByte(prompt[i+1])
			i += 2
			continue
		}

		// Check for opening bracket of option group: [
		if prompt[i] == '[' {
			// Find closing bracket ] that is not escaped
			start := i + 1
			closeIdx := -1
			j := start
			for j < n {
				if prompt[j] == '\\' && j+1 < n && (prompt[j+1] == '[' || prompt[j+1] == ']') {
					j += 2
					continue
				}
				if prompt[j] == ']' {
					closeIdx = j
					break
				}
				j++
			}

			if closeIdx != -1 {
				// We found a bracketed option group
				if currentLit.Len() > 0 {
					segments = append(segments, promptSegment{
						kind:    segmentLiteral,
						literal: currentLit.String(),
					})
					currentLit.Reset()
				}

				groupContent := prompt[start:closeIdx]
				rawOpts := strings.Split(groupContent, "|")
				var opts []string
				for _, raw := range rawOpts {
					// Handle escaped characters within options if any
					opts = append(opts, unescapeOption(raw))
				}

				segments = append(segments, promptSegment{
					kind:    segmentOptions,
					options: opts,
				})

				i = closeIdx + 1
				continue
			}
		}

		currentLit.WriteByte(prompt[i])
		i++
	}

	if currentLit.Len() > 0 {
		segments = append(segments, promptSegment{
			kind:    segmentLiteral,
			literal: currentLit.String(),
		})
	}

	return segments, nil
}

func unescapeOption(s string) string {
	var sb strings.Builder
	i := 0
	n := len(s)
	for i < n {
		if s[i] == '\\' && i+1 < n && (s[i+1] == '[' || s[i+1] == ']' || s[i+1] == '|') {
			sb.WriteByte(s[i+1])
			i += 2
			continue
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}
