package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

type Handler struct {
	service Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "email, password, and full_name are required")
		return
	}
	res, err := h.service.Register(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			response.Error(w, http.StatusConflict, response.CodeConflict, "email already registered")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "registration failed")
		return
	}
	response.JSON(w, http.StatusCreated, res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "email and password are required")
		return
	}
	res, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid credentials")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "login failed")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "refresh_token is required")
		return
	}
	res, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired token")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "token refresh failed")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "refresh_token is required")
		return
	}
	// Idempotent — ignore service errors (token may already be revoked)
	_ = h.service.Logout(r.Context(), req.RefreshToken)
	response.NoContent(w)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	res, err := h.service.Me(r.Context(), identity.UserID, identity.OrgID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to fetch user")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Email = ""
	}
	// Always 200 — prevents email enumeration
	_ = h.service.ForgotPassword(r.Context(), req.Email)
	response.JSON(w, http.StatusOK, map[string]string{
		"message": "if that email is registered, a reset link has been sent",
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.Token == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "token and password are required")
		return
	}
	if err := h.service.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		if errors.Is(err, ErrInvalidToken) {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired reset token")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "password reset failed")
		return
	}
	response.NoContent(w)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	identity, ok := IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "invalid request body")
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "current_password and new_password are required")
		return
	}
	if err := h.service.ChangePassword(r.Context(), identity.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, ErrWrongPassword) {
			response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "current password is incorrect")
		} else {
			response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to update password")
		}
		return
	}
	response.NoContent(w)
}
