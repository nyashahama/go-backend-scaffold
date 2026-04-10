package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_EmitsCredentialHeadersOnlyForAllowedOrigins(t *testing.T) {
	handler := CORS([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("allow credentials=%q, want empty", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow origin=%q, want empty", got)
	}
}

func TestCORS_EmitsHeadersForAllowedOrigins(t *testing.T) {
	handler := CORS([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials=%q, want true", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("allow origin=%q, want https://app.example.com", got)
	}
}
