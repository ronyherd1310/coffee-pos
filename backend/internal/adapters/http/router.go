package http

import (
	"encoding/json"
	"net/http"

	appauth "coffee-pos/backend/internal/app/auth"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type CookieConfig struct {
	Name     string
	Path     string
	Secure   bool
	SameSite http.SameSite
}

type RouterOptions struct {
	AuthService *appauth.Service
	Cookie      CookieConfig
}

func NewRouter(options ...RouterOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)

	if len(options) > 0 {
		cookieConfig := normalizeCookieConfig(options[0].Cookie)
		authHandlers := newAuthHandlers(options[0].AuthService, cookieConfig)
		authGuard := newAuthMiddleware(options[0].AuthService, cookieConfig.Name)
		mux.HandleFunc("POST /api/auth/login", authHandlers.handleLogin)
		mux.HandleFunc("POST /api/auth/logout", authHandlers.handleLogout)
		mux.HandleFunc("GET /api/auth/session", authHandlers.handleSession)
		mux.Handle("GET /api/pos/ping", authGuard.requireAuth(http.HandlerFunc(handleProtectedPOSPing)))
	}

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Service: "coffee-pos-backend",
	})
}

func handleProtectedPOSPing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
