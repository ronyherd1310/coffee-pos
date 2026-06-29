package http

import (
	"net/http"

	appauth "coffee-pos/backend/internal/app/auth"
)

type authMiddleware struct {
	service    *appauth.Service
	cookieName string
}

func newAuthMiddleware(service *appauth.Service, cookieName string) authMiddleware {
	return authMiddleware{
		service:    service,
		cookieName: cookieName,
	}
}

func (m authMiddleware) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.service == nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		result, err := m.service.Session(r.Context(), sessionIDFromRequest(r, m.cookieName))
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if !result.Authenticated {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}
