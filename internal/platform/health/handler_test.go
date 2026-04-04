package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type okChecker struct{}

func (o *okChecker) Ping(_ context.Context) error { return nil }

type failChecker struct{}

func (f *failChecker) Ping(_ context.Context) error { return errors.New("down") }

func TestHealthz_AlwaysOK(t *testing.T) {
	h := New(&okChecker{}, &okChecker{})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.Healthz(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestReadyz_AllHealthy(t *testing.T) {
	h := New(&okChecker{}, &okChecker{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	h.Readyz(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestReadyz_DBUnhealthy(t *testing.T) {
	h := New(&failChecker{}, &okChecker{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	h.Readyz(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestReadyz_CacheUnhealthy(t *testing.T) {
	h := New(&okChecker{}, &failChecker{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	h.Readyz(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}
