package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	appauth "coffee-pos/backend/internal/app/auth"
)

const loginBodyLimit = 1024

type authHandlers struct {
	service *appauth.Service
	cookie  CookieConfig
}

func newAuthHandlers(service *appauth.Service, cookie CookieConfig) authHandlers {
	return authHandlers{
		service: service,
		cookie:  cookie,
	}
}

func normalizeCookieConfig(config CookieConfig) CookieConfig {
	if config.Name == "" {
		config.Name = "coffee_pos_session"
	}
	if config.Path == "" {
		config.Path = "/"
	}
	if config.SameSite == 0 {
		config.SameSite = http.SameSiteLaxMode
	}
	return config
}

func (h authHandlers) handleLogin(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	pin, ok := readLoginPIN(w, r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "invalid_pin")
		return
	}

	result, err := h.service.Login(r.Context(), appauth.LoginInput{
		PIN:              pin,
		ClientID:         clientIdentifier(r),
		CurrentSessionID: sessionIDFromRequest(r, h.cookie.Name),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	switch result.Status {
	case appauth.LoginStatusInvalidPIN:
		writeJSONError(w, http.StatusUnauthorized, "invalid_pin")
	case appauth.LoginStatusTooManyAttempts:
		writeJSONError(w, http.StatusTooManyRequests, "too_many_attempts")
	case appauth.LoginStatusAuthenticated:
		http.SetCookie(w, buildSessionCookie(h.cookie, result.Session))
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
	}
}

func (h authHandlers) handleLogout(w http.ResponseWriter, r *http.Request) {
	if h.service != nil {
		if err := h.service.Logout(r.Context(), sessionIDFromRequest(r, h.cookie.Name)); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
			return
		}
	}

	http.SetCookie(w, clearSessionCookie(h.cookie))
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func (h authHandlers) handleSession(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
		return
	}

	result, err := h.service.Session(r.Context(), sessionIDFromRequest(r, h.cookie.Name))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": result.Authenticated})
}

func readLoginPIN(w http.ResponseWriter, r *http.Request) (string, bool) {
	reader := http.MaxBytesReader(w, r.Body, loginBodyLimit)
	defer reader.Close()

	var payload map[string]any
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return "", false
		}
		return "", false
	}

	pin, ok := payload["pin"].(string)
	if !ok {
		return "", false
	}

	return pin, true
}

func buildSessionCookie(config CookieConfig, session appauth.Session) *http.Cookie {
	return &http.Cookie{
		Name:     config.Name,
		Value:    session.ID,
		Path:     config.Path,
		HttpOnly: true,
		Secure:   config.Secure,
		SameSite: config.SameSite,
		Expires:  session.ExpiresAt,
	}
}

func clearSessionCookie(config CookieConfig) *http.Cookie {
	return &http.Cookie{
		Name:     config.Name,
		Value:    "",
		Path:     config.Path,
		HttpOnly: true,
		Secure:   config.Secure,
		SameSite: config.SameSite,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	}
}

func sessionIDFromRequest(r *http.Request, cookieName string) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func clientIdentifier(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
