package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lexis-app/lexis-api/internal/modules/progress/domain"
)

func TestMode_IsValid(t *testing.T) {
	tests := []struct {
		mode domain.Mode
		want bool
	}{
		{domain.ModeChat, true},
		{domain.ModeQuiz, true},
		{domain.ModeTranslate, true},
		{domain.ModeGap, true},
		{domain.ModeScramble, true},
		{"invalid", false},
		{"", false},
		{"CHAT", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.mode.IsValid())
		})
	}
}
