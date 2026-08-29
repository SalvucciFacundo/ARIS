package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"

	"github.com/google/uuid"
)

var _ ports.LLMProvider = (*OpenAIClient)(nil)

// OpenAIClient communicates with OpenAI-compatible chat completion endpoints.
type OpenAIClient struct {
	name       string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI compatible LLM provider.
func NewOpenAIClient(name, apiKey, baseURL, model string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIClient{
		name:    name,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (c *OpenAIClient) Name() string {
	return c.name
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends a raw chat completion request.
func (c *OpenAIClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("llm error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

type reasonOutputJSON struct {
	EnhancedPrompt string  `json:"enhanced_prompt"`
	NegativePrompt string  `json:"negative_prompt"`
	AspectRatio    string  `json:"aspect_ratio"`
	Model          string  `json:"model"`
	CFGScale       float64 `json:"cfg_scale"`
	Steps          int     `json:"steps"`
	Seed           int64   `json:"seed"`
}

// ReasonPrompt interprets user intent, factors in Knowledge Graph facts, and outputs an ImageSpec.
func (c *OpenAIClient) ReasonPrompt(ctx context.Context, input string, facts []domain.KnowledgeFact) (*domain.ImageSpec, error) {
	systemPrompt := `You are ARIS (Autonomous Reasoner for Image System) - an expert Art Director, Photographer, and Prompt Engineer.
Your role: Convert natural language requests into production-grade image prompts and parameters.

Guidelines:
1. Translate concepts to descriptive, high-quality visual English.
2. Structure the positive prompt with subjects, environment, lighting, camera lens/angle, color palette, and rendering details.
3. Formulate appropriate negative prompts (e.g. artifacts, blur, bad anatomy, unwanted text).
4. Respect any recalled user preferences and stylistic facts provided.
5. Return ONLY a valid JSON object matching the schema:
{
  "enhanced_prompt": "detailed visual prompt in english",
  "negative_prompt": "negative prompt triggers",
  "aspect_ratio": "1:1" | "16:9" | "9:16" | "4:3" | "3:4" | "21:9",
  "model": "flux",
  "cfg_scale": 7.0,
  "steps": 25,
  "seed": 0
}`

	var factSummary strings.Builder
	if len(facts) > 0 {
		factSummary.WriteString("\n\n### RECALLED USER FACTS & STYLES:\n")
		for _, f := range facts {
			factSummary.WriteString(fmt.Sprintf("- [%s] %s: %s\n", f.Scope, f.Topic, f.Fact))
		}
	}

	userMessage := fmt.Sprintf("User Request: %q%s", input, factSummary.String())

	rawJSON, err := c.Complete(ctx, systemPrompt, userMessage)
	if err != nil {
		return nil, fmt.Errorf("llm prompt reasoning failed: %w", err)
	}

	// Clean code blocks if LLM returned ```json ... ```
	cleanedJSON := cleanJSONBlock(rawJSON)

	var out reasonOutputJSON
	if err := json.Unmarshal([]byte(cleanedJSON), &out); err != nil {
		// Fallback if model failed to return strict JSON
		return &domain.ImageSpec{
			ID:             uuid.New().String(),
			RawPrompt:      input,
			EnhancedPrompt: rawJSON,
			AspectRatio:    domain.RatioSquare,
			Width:          1024,
			Height:         1024,
			Steps:          20,
			CFGScale:       7.0,
			Backend:        "pollinations",
			Model:          "flux",
			CreatedAt:      time.Now(),
		}, nil
	}

	ratio := domain.ParseAspectRatio(out.AspectRatio)
	w, h := ratio.Dimensions(1024)

	steps := out.Steps
	if steps <= 0 {
		steps = 20
	}
	cfg := out.CFGScale
	if cfg <= 0 {
		cfg = 7.0
	}
	model := out.Model
	if model == "" {
		model = "flux"
	}

	return &domain.ImageSpec{
		ID:             uuid.New().String(),
		RawPrompt:      input,
		EnhancedPrompt: out.EnhancedPrompt,
		NegativePrompt: out.NegativePrompt,
		AspectRatio:    ratio,
		Width:          w,
		Height:         h,
		Steps:          steps,
		CFGScale:       cfg,
		Seed:           out.Seed,
		Backend:        "pollinations",
		Model:          model,
		CreatedAt:      time.Now(),
	}, nil
}

func cleanJSONBlock(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
