package handler

import (
	"net/http"

	"github.com/lexis-app/lexis-api/internal/shared/httputil"
)

// AIModel describes an AI model available in the system.
type AIModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Provider    string `json:"provider"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
}

var availableModels = []AIModel{
	{ID: "claude-sonnet-4-20250514", DisplayName: "Claude Sonnet", Provider: "anthropic", Icon: "A", Description: "Лучшие объяснения", Available: true},
	{ID: "claude-haiku-4-20250514", DisplayName: "Claude Haiku", Provider: "anthropic", Icon: "A", Description: "Быстрый для квизов", Available: true},
	{ID: "qwen-plus", DisplayName: "Qwen Plus", Provider: "qwen", Icon: "Q", Description: "Азиатские языки", Available: true},
	{ID: "gpt-4o", DisplayName: "GPT-4o", Provider: "openai", Icon: "G", Description: "Широкая совместимость", Available: true},
	{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Provider: "openai", Icon: "G", Description: "Экономичный", Available: true},
	{ID: "gemini-2.0-flash", DisplayName: "Gemini Flash", Provider: "google", Icon: "✦", Description: "Скорость + контекст", Available: true},
}

// HandleGetModels handles GET /api/v1/models and returns all available AI models.
func HandleGetModels(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string][]AIModel{"models": availableModels})
}
