package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/handler"
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

func decodeModels(t *testing.T, rec *httptest.ResponseRecorder) []usecase.CatalogModel {
	t.Helper()
	var resp struct {
		Models []usecase.CatalogModel `json:"models"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	return resp.Models
}

func TestHandleListOpenRouterModels_Success(t *testing.T) {
	src := stubCatalogSource{models: []usecase.RawCatalogModel{
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", InputModalities: []string{"text"}, OutputModalities: []string{"text"}},
	}}
	h := handler.NewModelsHandler(usecase.NewModelCatalogService(src))

	req := httptest.NewRequest(http.MethodGet, "/ai/models/openrouter", nil)
	rec := httptest.NewRecorder()
	h.HandleListOpenRouterModels(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	models := decodeModels(t, rec)
	require.Len(t, models, 1)
	assert.Equal(t, "openai/gpt-4o-mini", models[0].ID)
}

func TestHandleListOpenRouterModels_FallbackOnError(t *testing.T) {
	src := stubCatalogSource{err: errors.New("upstream down")}
	h := handler.NewModelsHandler(usecase.NewModelCatalogService(src))

	req := httptest.NewRequest(http.MethodGet, "/ai/models/openrouter", nil)
	rec := httptest.NewRecorder()
	h.HandleListOpenRouterModels(rec, req)

	// Graceful: the UI still gets a usable list even when OpenRouter is down.
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, decodeModels(t, rec))
}
