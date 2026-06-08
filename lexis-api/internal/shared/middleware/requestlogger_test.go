package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexis-app/lexis-api/internal/shared/logging"
	mw "github.com/lexis-app/lexis-api/internal/shared/middleware"
)

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, "info", "production")

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	handler := mw.RequestLogger(logger)(inner)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/foo", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.NotEmpty(t, buf.String(), "request logger must emit a line")
	var record map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &record))

	assert.Equal(t, "GET", record["method"])
	assert.Equal(t, "/foo", record["path"])
	assert.Equal(t, float64(http.StatusCreated), record["status"])
	assert.Contains(t, record, "duration_ms")
}
