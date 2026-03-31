package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lexis-app/lexis-api/internal/modules/progress/domain"
)

func TestUpdateGoalProgress_NoError(t *testing.T) {
	goals := []domain.Goal{
		{Name: "Test1", Progress: 50, Color: "amber"},
		{Name: "Test2", Progress: 30, Color: "red"},
	}
	updated := domain.UpdateGoalProgress(goals, false)
	assert.Equal(t, 53, updated[0].Progress)
	assert.Equal(t, 33, updated[1].Progress)
	assert.Equal(t, "amber", updated[0].Color)
	assert.Equal(t, "red", updated[1].Color)
}

func TestUpdateGoalProgress_HasError(t *testing.T) {
	goals := []domain.Goal{
		{Name: "Strong", Progress: 80, Color: "green"},
		{Name: "Weak", Progress: 10, Color: "red"},
	}
	updated := domain.UpdateGoalProgress(goals, true)
	// Weakest goal gets +8
	assert.Equal(t, 80, updated[0].Progress)
	assert.Equal(t, 18, updated[1].Progress)
}

func TestUpdateGoalProgress_ColorThresholds(t *testing.T) {
	goals := []domain.Goal{
		{Name: "Low", Progress: 37, Color: "red"},
	}
	updated := domain.UpdateGoalProgress(goals, false)
	assert.Equal(t, 40, updated[0].Progress)
	assert.Equal(t, "amber", updated[0].Color)

	updated = domain.UpdateGoalProgress(updated, false)
	assert.Equal(t, 43, updated[0].Progress)
	assert.Equal(t, "amber", updated[0].Color)
}

func TestUpdateGoalProgress_Cap100(t *testing.T) {
	goals := []domain.Goal{
		{Name: "Almost", Progress: 99, Color: "green"},
	}
	updated := domain.UpdateGoalProgress(goals, false)
	assert.Equal(t, 100, updated[0].Progress)
}
