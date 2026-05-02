package middleware

import (
	"net/http"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

// OrgIDFromRequest extracts an org ID from a request, usually from a route parameter.
type OrgIDFromRequest func(r *http.Request) (string, bool)

// RequireRole allows only authenticated identities with one of the allowed roles.
func RequireRole(roles ...auth.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := auth.IdentityFromRequest(r)
			if !ok {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing authenticated identity")
				return
			}
			if !auth.HasAnyRole(identity, roles...) {
				response.Error(w, http.StatusForbidden, response.CodeForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrgAccess allows only authenticated identities scoped to the requested org.
func RequireOrgAccess(orgIDFromRequest OrgIDFromRequest) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := auth.IdentityFromRequest(r)
			if !ok {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing authenticated identity")
				return
			}

			orgID, ok := orgIDFromRequest(r)
			if !ok || orgID == "" {
				response.Error(w, http.StatusBadRequest, response.CodeBadRequest, "missing org id")
				return
			}
			if !auth.CanAccessOrg(identity, orgID) {
				response.Error(w, http.StatusForbidden, response.CodeForbidden, "cannot access requested org")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
