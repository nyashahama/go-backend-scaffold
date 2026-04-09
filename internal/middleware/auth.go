package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	dbgen "github.com/nyashahama/go-backend-scaffold/db/gen"
	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

type UserReader interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (dbgen.User, error)
}

// Auth validates the Bearer JWT and injects identity into the request context.
// Context keys are defined in the auth package to avoid import cycles.
func Auth(jwtSecret string, users UserReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid authorization header format")
				return
			}

			tokenStr := parts[1]
			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing token")
				return
			}

			claims, err := auth.ValidateAccessToken(tokenStr, jwtSecret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired token")
				return
			}

			if users != nil {
				userID, err := uuid.Parse(claims.Subject)
				if err != nil {
					response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired token")
					return
				}
				user, err := users.GetUserByID(r.Context(), userID)
				if err != nil || user.TokenVersion != claims.TokenVersion {
					response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "invalid or expired token")
					return
				}
			}

			next.ServeHTTP(w, r.WithContext(auth.ContextWithClaims(r.Context(), claims)))
		})
	}
}
