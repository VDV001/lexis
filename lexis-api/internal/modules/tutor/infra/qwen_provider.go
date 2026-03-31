package infra

const qwenBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"

// QwenProvider wraps OpenAICompatibleProvider with the Qwen (Alibaba) base URL.
type QwenProvider struct {
	*OpenAICompatibleProvider
}

func NewQwenProvider(apiKey string) *QwenProvider {
	return &QwenProvider{
		OpenAICompatibleProvider: NewOpenAICompatibleProvider(apiKey, qwenBaseURL),
	}
}
