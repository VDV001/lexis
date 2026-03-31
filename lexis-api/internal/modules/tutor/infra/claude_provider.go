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

	"github.com/lexis-app/lexis-api/internal/modules/tutor/domain"
)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"

type ClaudeProvider struct {
	apiKey string
	client *http.Client
}

func NewClaudeProvider(apiKey string) *ClaudeProvider {
	return &ClaudeProvider{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (p *ClaudeProvider) Chat(ctx context.Context, req domain.ChatRequest) (<-chan domain.ChatDelta, error) {
	ch := make(chan domain.ChatDelta, 32)

	messages := make([]map[string]string, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = map[string]string{"role": m.Role, "content": m.Content}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"system":     req.System,
		"messages":   messages,
		"stream":     true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq) //nolint:bodyclose // closed in goroutine below
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(errBody))
	}

	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()
		p.streamResponse(ctx, resp.Body, ch)
	}()

	return ch, nil
}

func (p *ClaudeProvider) streamResponse(ctx context.Context, body io.Reader, ch chan<- domain.ChatDelta) {
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
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}

		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		if event.Type == "content_block_delta" && event.Delta.Text != "" {
			fullText.WriteString(event.Delta.Text)
			ch <- domain.ChatDelta{Type: "delta", Content: event.Delta.Text}
		}

		if event.Type == "message_stop" {
			break
		}
	}

	// After streaming, try to parse the full response as JSON for structured data
	text := fullText.String()
	p.parseStructuredResponse(text, ch)

	ch <- domain.ChatDelta{Type: "done"}
}

func (p *ClaudeProvider) parseStructuredResponse(text string, ch chan<- domain.ChatDelta) {
	// Try to parse as the expected JSON format
	var resp struct {
		Reply      string             `json:"reply"`
		Correction *domain.Correction `json:"correction"`
		Feedback   *domain.Feedback   `json:"feedback"`
		ErrorType  *string            `json:"error_type"`
		NewWords   []string           `json:"new_words"`
	}

	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return
	}

	if resp.Correction != nil {
		ch <- domain.ChatDelta{Type: "correction", Correction: resp.Correction}
	}
	if resp.Feedback != nil {
		ch <- domain.ChatDelta{Type: "feedback", Feedback: resp.Feedback}
	}
	if len(resp.NewWords) > 0 {
		ch <- domain.ChatDelta{Type: "words", Words: resp.NewWords}
	}
}

func (p *ClaudeProvider) GenerateExercise(ctx context.Context, req domain.ExerciseRequest) (domain.Exercise, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"system":     req.System,
		"messages":   []map[string]string{{"role": "user", "content": "Generate"}},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.Exercise{}, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.Exercise{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Exercise{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return domain.Exercise{}, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.Exercise{}, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Content) == 0 {
		return domain.Exercise{}, fmt.Errorf("empty response")
	}

	return domain.Exercise{Raw: result.Content[0].Text}, nil
}

func (p *ClaudeProvider) CheckAnswer(ctx context.Context, req domain.CheckRequest) (domain.CheckResult, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	body := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"system":     req.System,
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Exercise: %s\nUser answer: %s", req.Context, req.UserAnswer)},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return domain.CheckResult{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(jsonBody))
	if err != nil {
		return domain.CheckResult{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.CheckResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return domain.CheckResult{}, fmt.Errorf("anthropic error %d: %s", resp.StatusCode, string(errBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return domain.CheckResult{}, err
	}

	if len(result.Content) == 0 {
		return domain.CheckResult{}, fmt.Errorf("empty response")
	}

	return domain.CheckResult{Raw: result.Content[0].Text}, nil
}
