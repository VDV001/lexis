package infra

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
)

const geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

type GeminiProvider struct {
	apiKey string
	client *http.Client
}

func NewGeminiProvider(apiKey string) *GeminiProvider {
	return &GeminiProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// geminiContent represents a content block in the Gemini API format.
type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

func (p *GeminiProvider) Chat(ctx context.Context, req domain.ChatRequest) (<-chan domain.ChatDelta, error) {
	ch := make(chan domain.ChatDelta, 32)

	contents := p.buildContents(req.System, req.Messages)

	body := map[string]interface{}{
		"contents": contents,
	}

	// SECURITY: The API key is passed as a URL query parameter (required by Google's
	// Gemini REST API). Never log or include this URL in error messages, as it would
	// leak the key. All error paths below use only resp.StatusCode and resp.Body.
	url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse&key=%s", geminiBaseURL, req.Model, p.apiKey)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq) //nolint:bodyclose // closed in goroutine below
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("gemini API error %d: %s", resp.StatusCode, string(errBody))
	}

	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()
		p.streamResponse(ctx, resp.Body, ch)
	}()

	return ch, nil
}

func (p *GeminiProvider) streamResponse(ctx context.Context, body io.Reader, ch chan<- domain.ChatDelta) {
	scanner := bufio.NewScanner(body)
	var fullText strings.Builder

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
			text := event.Candidates[0].Content.Parts[0].Text
			if text != "" {
				fullText.WriteString(text)
				if !sendDelta(ctx, ch, domain.ChatDelta{Type: "delta", Content: text}) {
					return
				}
			}
		}
	}

	// After streaming, try to parse the full response as JSON for structured data
	text := fullText.String()
	parseStructuredResponse(ctx, text, ch)

	sendDelta(ctx, ch, domain.ChatDelta{Type: "done"})
}

func (p *GeminiProvider) buildContents(system string, messages []domain.Message) []geminiContent {
	contents := make([]geminiContent, 0, len(messages)+1)

	// Gemini uses a "user" message at the start for system instructions
	if system != "" {
		contents = append(contents, geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: system}},
		})
		contents = append(contents, geminiContent{
			Role:  "model",
			Parts: []geminiPart{{Text: "Understood. I will follow these instructions."}},
		})
	}

	for _, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	return contents
}

func (p *GeminiProvider) GenerateExercise(ctx context.Context, req domain.ExerciseRequest) (domain.Exercise, error) {
	contents := p.buildContents(req.System, []domain.Message{
		{Role: "user", Content: "Generate"},
	})

	body := map[string]interface{}{
		"contents": contents,
	}

	// SECURITY: API key in URL — see comment in Chat method.
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, req.Model, p.apiKey)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.Exercise{}, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.Exercise{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Exercise{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return domain.Exercise{}, fmt.Errorf("gemini error %d: %s", resp.StatusCode, string(errBody))
	}

	text, err := p.extractText(resp.Body)
	if err != nil {
		return domain.Exercise{}, err
	}

	return domain.Exercise{Raw: text}, nil
}

func (p *GeminiProvider) CheckAnswer(ctx context.Context, req domain.CheckRequest) (domain.CheckResult, error) {
	contents := p.buildContents(req.System, []domain.Message{
		{Role: "user", Content: fmt.Sprintf("Exercise: %s\nUser answer: %s", req.Context, req.UserAnswer)},
	})

	body := map[string]interface{}{
		"contents": contents,
	}

	// SECURITY: API key in URL — see comment in Chat method.
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiBaseURL, req.Model, p.apiKey)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.CheckResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.CheckResult{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.CheckResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return domain.CheckResult{}, fmt.Errorf("gemini error %d: %s", resp.StatusCode, string(errBody))
	}

	text, err := p.extractText(resp.Body)
	if err != nil {
		return domain.CheckResult{}, err
	}

	return domain.CheckResult{Raw: text}, nil
}

func (p *GeminiProvider) extractText(body io.Reader) (string, error) {
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}
