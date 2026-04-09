package httputil_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexis-app/lexis-api/internal/shared/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteProblem(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.WriteProblem(w, http.StatusNotFound, "Not found", "user not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var body httputil.ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Not found", body.Title)
	assert.Equal(t, "user not found", body.Detail)
	assert.Equal(t, 404, body.Status)
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	payload := map[string]string{"hello": "world"}
	httputil.WriteJSON(w, http.StatusCreated, payload)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "world", body["hello"])
}

func TestWriteJSON_NilPayload(t *testing.T) {
	w := httptest.NewRecorder()
	httputil.WriteJSON(w, http.StatusOK, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "null\n", w.Body.String())
}
