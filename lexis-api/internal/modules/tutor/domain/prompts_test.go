package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt(t *testing.T) {
	settings := PromptSettings{
		UserName:         "Test User",
		ProficiencyLevel: "b1",
		VocabularyType:   "tech",
	}

	tests := []struct {
		mode     Mode
		contains string
	}{
		{ModeChat, "Test User"},
		{ModeQuiz, "grammar/vocabulary question"},
		{ModeTranslate, "Russian"},
		{ModeGap, "gap"},
		{ModeScramble, "scrambl"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			prompt := BuildSystemPrompt(settings, tt.mode)
			assert.NotEmpty(t, prompt)
			assert.Contains(t, prompt, tt.contains)
		})
	}

	t.Run("unknown mode falls back to chat", func(t *testing.T) {
		prompt := BuildSystemPrompt(settings, "unknown")
		assert.Contains(t, prompt, "Test User")
	})

	t.Run("unknown level defaults to b1", func(t *testing.T) {
		s := settings
		s.ProficiencyLevel = "z9"
		prompt := BuildSystemPrompt(s, ModeChat)
		assert.NotEmpty(t, prompt)
	})

	t.Run("unknown vocab type defaults to tech", func(t *testing.T) {
		s := settings
		s.VocabularyType = "nonexistent"
		prompt := BuildSystemPrompt(s, ModeChat)
		assert.NotEmpty(t, prompt)
	})
}
