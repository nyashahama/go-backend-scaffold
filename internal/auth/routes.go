package auth

import "github.com/go-chi/chi/v5"

// Routes returns public auth endpoints (no JWT required).
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	return r
}

// ProtectedRoutes returns JWT-required auth endpoints.
func (h *Handler) ProtectedRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/me", h.Me)
	r.Post("/change-password", h.ChangePassword)
	return r
}
