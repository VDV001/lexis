package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/lexis-app/lexis-api/internal/modules/tutor/usecase"
	"github.com/lexis-app/lexis-api/internal/shared/httputil"
)

// modelCatalog is the port the handler consumes to list selectable models.
// Defined here (the consumer) per the dependency-inversion convention.
type modelCatalog interface {
	List(ctx context.Context) ([]usecase.CatalogModel, error)
}

// ModelsHandler serves the dynamic OpenRouter model catalogue.
type ModelsHandler struct {
	catalog modelCatalog
}

func NewModelsHandler(catalog modelCatalog) *ModelsHandler {
	return &ModelsHandler{catalog: catalog}
}

// HandleListOpenRouterModels handles GET /api/v1/ai/models/openrouter. It always
// responds 200 with a usable list: on upstream failure the service returns an
// embedded fallback shortlist, and the error is logged for observability.
func (h *ModelsHandler) HandleListOpenRouterModels(w http.ResponseWriter, r *http.Request) {
	models, err := h.catalog.List(r.Context())
	if err != nil {
		slog.Warn("openrouter catalog: serving fallback after upstream error", "error", err)
	}
	httputil.WriteJSON(w, http.StatusOK, map[string][]usecase.CatalogModel{"models": models})
}
