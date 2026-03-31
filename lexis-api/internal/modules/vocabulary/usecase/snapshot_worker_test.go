package usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lexis-app/lexis-api/internal/modules/vocabulary/usecase"
)

func TestNewSnapshotTask(t *testing.T) {
	task := usecase.NewSnapshotTask()
	assert.Equal(t, usecase.TaskVocabSnapshot, task.Type())
}
