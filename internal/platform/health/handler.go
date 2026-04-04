package health

import (
	"context"
	"net/http"
	"time"

	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

// Checker is implemented by anything that can confirm it is reachable.
type Checker interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db    Checker
	cache Checker
}

func New(db Checker, cache Checker) *Handler {
	return &Handler{db: db, cache: cache}
}

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := map[string]string{}
	healthy := true

	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["database"] = err.Error()
			healthy = false
		} else {
			checks["database"] = "ok"
		}
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			checks["cache"] = err.Error()
			healthy = false
		} else {
			checks["cache"] = "ok"
		}
	}

	if !healthy {
		response.Error(w, http.StatusServiceUnavailable, "NOT_READY", "one or more dependencies are unhealthy")
		return
	}

	response.JSON(w, http.StatusOK, checks)
}
