package exampledomain

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nyashahama/go-backend-scaffold/internal/middleware"
)

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/orgs/{orgID}/resources", func(r chi.Router) {
		r.Use(middleware.RequireOrgAccess(orgIDFromRoute))
		r.Get("/", h.ListResources)
	})
}

func orgIDFromRoute(r *http.Request) (string, bool) {
	orgID := chi.URLParam(r, "orgID")
	return orgID, orgID != ""
}
