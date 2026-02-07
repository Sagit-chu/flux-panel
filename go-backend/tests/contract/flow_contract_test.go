package contract_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-backend/internal/http/handler"
)

func TestFlowEndpointsStringResponses(t *testing.T) {
	h := handler.New(nil, "secret")
	mux := http.NewServeMux()
	h.Register(mux)

	tests := []struct {
		name     string
		method   string
		path     string
		expected string
	}{
		{name: "flow test", method: http.MethodGet, path: "/flow/test", expected: "test"},
		{name: "flow config", method: http.MethodPost, path: "/flow/config?secret=abc", expected: "ok"},
		{name: "flow upload", method: http.MethodPost, path: "/flow/upload?secret=abc", expected: "ok"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			if string(body) != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, string(body))
			}
		})
	}
}
