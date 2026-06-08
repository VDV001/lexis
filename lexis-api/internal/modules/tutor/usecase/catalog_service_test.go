package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCatalogSource struct {
	models []usecase.RawCatalogModel
	err    error
}

func (s stubCatalogSource) List(context.Context) ([]usecase.RawCatalogModel, error) {
	return s.models, s.err
}

func TestModelCatalogService_Success(t *testing.T) {
	src := stubCatalogSource{models: []usecase.RawCatalogModel{
		{ID: "openai/gpt-4o-mini", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
	}}
	svc := usecase.NewModelCatalogService(src)

	models, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "openai/gpt-4o-mini", models[0].ID)
}

func TestModelCatalogService_FetchError_ReturnsFallback(t *testing.T) {
	src := stubCatalogSource{err: errors.New("network down")}
	svc := usecase.NewModelCatalogService(src)

	models, err := svc.List(context.Background())
	require.Error(t, err, "the upstream failure must be surfaced for logging")
	assert.NotEmpty(t, models, "UI must still receive a usable fallback list")
}

func TestModelCatalogService_EmptyAfterFilter_ReturnsFallback(t *testing.T) {
	// Source returns only non-chat / non-curated models -> filter yields nothing.
	src := stubCatalogSource{models: []usecase.RawCatalogModel{
		{ID: "randomlab/foo", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
	}}
	svc := usecase.NewModelCatalogService(src)

	models, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, models, "an empty filtered result must fall back to the embedded shortlist")
}
