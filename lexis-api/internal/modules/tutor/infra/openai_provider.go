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

// OpenAICompatibleProvider works with any OpenAI-compatible Chat Completions API
// (OpenAI, Qwen, etc.) by accepting a configurable base URL.
type OpenAICompatibleProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAICompatibleProvider(apiKey, baseURL string) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *OpenAICompatibleProvider) Chat(ctx context.Context, req domain.ChatRequest) (<-chan domain.ChatDelta, error) {
	ch := make(chan domain.ChatDelta, 32)

	messages := make([]map[string]string, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, map[string]string{"role": m.Role, "content": m.Content})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"messages":   messages,
		"stream":     true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq) //nolint:bodyclose // closed in goroutine below
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("openai-compatible API error %d: %s", resp.StatusCode, string(errBody))
	}

	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()
		p.streamResponse(ctx, resp.Body, ch)
	}()

	return ch, nil
}

func (p *OpenAICompatibleProvider) streamResponse(ctx context.Context, body io.Reader, ch chan<- domain.ChatDelta) {
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
		if data == "[DONE]" {
			break
		}

		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			text := event.Choices[0].Delta.Content
			fullText.WriteString(text)
			if !sendDelta(ctx, ch, domain.ChatDelta{Type: "delta", Content: text}) {
				return
			}
		}
	}

	// After streaming, try to parse the full response as JSON for structured data
	text := fullText.String()
	parseStructuredResponse(ctx, text, ch)

	sendDelta(ctx, ch, domain.ChatDelta{Type: "done"})
}

func (p *OpenAICompatibleProvider) GenerateExercise(ctx context.Context, req domain.ExerciseRequest) (domain.Exercise, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	messages := []map[string]string{
		{"role": "system", "content": req.System},
		{"role": "user", "content": "Generate"},
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.Exercise{}, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.Exercise{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Exercise{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return domain.Exercise{}, fmt.Errorf("openai-compatible error %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.Exercise{}, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return domain.Exercise{}, fmt.Errorf("empty response")
	}

	return domain.Exercise{Raw: result.Choices[0].Message.Content}, nil
}

func (p *OpenAICompatibleProvider) CheckAnswer(ctx context.Context, req domain.CheckRequest) (domain.CheckResult, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	messages := []map[string]string{
		{"role": "system", "content": req.System},
		{"role": "user", "content": fmt.Sprintf("Exercise: %s\nUser answer: %s", req.Context, req.UserAnswer)},
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.CheckResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.CheckResult{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.CheckResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return domain.CheckResult{}, fmt.Errorf("openai-compatible error %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.CheckResult{}, err
	}

	if len(result.Choices) == 0 {
		return domain.CheckResult{}, fmt.Errorf("empty response")
	}

	return domain.CheckResult{Raw: result.Choices[0].Message.Content}, nil
}
