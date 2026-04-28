package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	mw "github.com/lexis-app/lexis-api/internal/shared/middleware"
)

func TestRequireJSON(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantCode    int
	}{
		{
			name:        "POST with application/json",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"key":"val"}`,
			wantCode:    http.StatusOK,
		},
		{
			name:        "POST with application/json charset",
			method:      http.MethodPost,
			contentType: "application/json; charset=utf-8",
			body:        `{"key":"val"}`,
			wantCode:    http.StatusOK,
		},
		{
			name:        "POST with text/plain rejected",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        "hello",
			wantCode:    http.StatusUnsupportedMediaType,
		},
		{
			name:        "PUT with form-urlencoded rejected",
			method:      http.MethodPut,
			contentType: "application/x-www-form-urlencoded",
			body:        "key=val",
			wantCode:    http.StatusUnsupportedMediaType,
		},
		{
			name:        "PATCH with multipart rejected",
			method:      http.MethodPatch,
			contentType: "multipart/form-data",
			body:        "data",
			wantCode:    http.StatusUnsupportedMediaType,
		},
		{
			name:        "POST without content-type header passes through",
			method:      http.MethodPost,
			contentType: "",
			body:        `{"key":"val"}`,
			wantCode:    http.StatusOK,
		},
		{
			name:        "GET without body passes through",
			method:      http.MethodGet,
			contentType: "",
			body:        "",
			wantCode:    http.StatusOK,
		},
		{
			name:        "DELETE without body passes through",
			method:      http.MethodDelete,
			contentType: "",
			body:        "",
			wantCode:    http.StatusOK,
		},
		{
			name:        "GET with content-length>0 and wrong type",
			method:      http.MethodGet,
			contentType: "text/plain",
			body:        "data",
			wantCode:    http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			handler := mw.RequireJSON(inner)

			var body *strings.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			} else {
				body = strings.NewReader("")
			}

			req := httptest.NewRequestWithContext(context.Background(), tc.method, "/test", body)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			// For GET requests with no body, set ContentLength = 0
			if tc.body == "" {
				req.ContentLength = 0
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, tc.wantCode, rec.Code)

			if tc.wantCode == http.StatusUnsupportedMediaType {
				assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			}
		})
	}
}
