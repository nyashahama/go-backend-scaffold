package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns public auth endpoints (no JWT required).
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	h.registerPublicRoutes(r)
	return r
}

// ProtectedRoutes returns JWT-required auth endpoints.
func (h *Handler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()
	h.registerProtectedRoutes(r)
	return r
}

// RegisterRoutes registers public and protected auth endpoints on the provided router.
func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	h.registerPublicRoutes(r)
	r.Group(func(r chi.Router) {
		if authMiddleware != nil {
			r.Use(authMiddleware)
		}
		h.registerProtectedRoutes(r)
	})
}

func (h *Handler) registerPublicRoutes(r chi.Router) {
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
}

func (h *Handler) registerProtectedRoutes(r chi.Router) {
	r.Get("/me", h.Me)
	r.Post("/change-password", h.ChangePassword)
}
