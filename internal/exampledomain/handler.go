package exampledomain

import (
	"errors"
	"net/http"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

type Handler struct {
	service Servicer
}

func NewHandler(service Servicer) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromRequest(r)
	if !ok {
		response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing auth context")
		return
	}

	resources, err := h.service.List(r.Context(), identity.OrgID)
	if err != nil {
		if errors.Is(err, ErrMissingOrgID) {
			response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "missing org id")
			return
		}
		response.Error(w, http.StatusInternalServerError, response.CodeInternalError, "failed to list resources")
		return
	}

	response.JSON(w, http.StatusOK, resources)
}
