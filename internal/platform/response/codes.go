package response

import "net/http"

const (
	CodeBadRequest    = "BAD_REQUEST"
	CodeConflict      = "CONFLICT"
	CodeForbidden     = "FORBIDDEN"
	CodeInternalError = "INTERNAL_ERROR"
	CodeNotFound      = "NOT_FOUND"
	CodeRateLimited   = "RATE_LIMITED"
	CodeUnauthorized  = "UNAUTHORIZED"
)

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
