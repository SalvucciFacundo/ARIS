package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
)

var _ ports.VisionCritic = (*VisionClient)(nil)

// VisionClient implements ports.VisionCritic using multi-modal OpenAI/Ollama endpoints.
type VisionClient struct {
	name       string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewVisionClient creates a new vision critic client.
func NewVisionClient(name, apiKey, baseURL, model string, httpClient *http.Client) *VisionClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &VisionClient{
		name:       name,
		apiKey:     apiKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: httpClient,
	}
}

func (v *VisionClient) Name() string {
	return v.name
}

type visionMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type imageContent struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type visionRequest struct {
	Model       string          `json:"model"`
	Messages    []visionMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type visionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type CritiqueReport struct {
	Score        float64 `json:"score"` // 0.0 to 1.0
	Adherence    string  `json:"adherence"`
	Defects      string  `json:"defects"`
	SuggestedFix string  `json:"suggested_fix"`
}

func (v *VisionClient) Evaluate(ctx context.Context, imagePath string, spec *domain.ImageSpec) (float64, string, error) {
	imgBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return 0, "", fmt.Errorf("read image file for vision critique: %w", err)
	}

	mimeType := "image/jpeg"
	ext := strings.ToLower(filepath.Ext(imagePath))
	if ext == ".png" {
		mimeType = "image/png"
	} else if ext == ".webp" {
		mimeType = "image/webp"
	}

	b64Data := base64.StdEncoding.EncodeToString(imgBytes)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64Data)

	systemPrompt := `You are an expert Vision QA & Art Director Critic for AI generated images.
Evaluate the provided image against the user's prompt specification.
Score the output between 0.0 (completely flawed / missed subject) to 1.0 (flawless masterpiece).

Check for:
1. Subject & prompt adherence: Are all requested elements present?
2. Visual quality: Artifacts, strange anatomy, blurry faces, unwanted text/watermarks.
3. Lighting, composition, and style fidelity.

Return ONLY a valid JSON object matching:
{
  "score": 0.85,
  "adherence": "High/Medium/Low with explanation",
  "defects": "Identified flaws or 'None'",
  "suggested_fix": "Concrete prompt keywords to fix defects or empty string"
}`

	userText := fmt.Sprintf("Target Prompt: %q\nNegative Prompt: %q\nAspect Ratio: %s",
		spec.EnhancedPrompt, spec.NegativePrompt, spec.AspectRatio)

	reqPayload := visionRequest{
		Model: v.model,
		Messages: []visionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: []any{
				textContent{Type: "text", Text: userText},
				imageContent{Type: "image_url", ImageURL: struct {
					URL string `json:"url"`
				}{URL: dataURL}},
			}},
		},
		Temperature: 0.2,
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return 0, "", fmt.Errorf("marshal vision payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", v.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, "", fmt.Errorf("create vision request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if v.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", v.apiKey))
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("vision request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("read vision response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("vision api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var parsed visionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return 0, "", fmt.Errorf("unmarshal vision response: %w", err)
	}

	if parsed.Error != nil {
		return 0, "", fmt.Errorf("vision error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return 0, "", fmt.Errorf("no choices in vision response")
	}

	rawContent := parsed.Choices[0].Message.Content
	cleaned := cleanJSON(rawContent)

	var report CritiqueReport
	if err := json.Unmarshal([]byte(cleaned), &report); err != nil {
		// Fallback if parsing failed
		return 0.75, fmt.Sprintf("Critique: %s", rawContent), nil
	}

	critiqueSummary := fmt.Sprintf("Score: %.2f | Adherence: %s | Defects: %s",
		report.Score, report.Adherence, report.Defects)
	if report.SuggestedFix != "" {
		critiqueSummary += fmt.Sprintf(" | Fix: %s", report.SuggestedFix)
	}

	return report.Score, critiqueSummary, nil
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
