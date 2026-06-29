package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProtectedRouteReturnsUnauthorizedWithoutSession(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	request := httptest.NewRequest(http.MethodGet, "/api/pos/ping", nil)
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedRouteAllowsAuthenticatedSession(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	loginResponse := fixture.login(t, `{"pin":"123456"}`, nil, "203.0.113.10:1234")
	cookie := assertSingleCookie(t, loginResponse, fixture.cookieName)

	request := httptest.NewRequest(http.MethodGet, "/api/pos/ping", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected protected route success, got %d", response.Code)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if body.Status != "ok" {
		t.Fatalf("expected status ok, got %q", body.Status)
	}
}

func TestProtectedRouteRejectsTamperedAndExpiredSessions(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	loginResponse := fixture.login(t, `{"pin":"123456"}`, nil, "203.0.113.10:1234")
	cookie := assertSingleCookie(t, loginResponse, fixture.cookieName)

	tamperedRequest := httptest.NewRequest(http.MethodGet, "/api/pos/ping", nil)
	tamperedRequest.AddCookie(&http.Cookie{Name: fixture.cookieName, Value: cookie.Value + "-tampered"})
	tamperedResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(tamperedResponse, tamperedRequest)
	assertJSONError(t, tamperedResponse, http.StatusUnauthorized, "unauthorized")

	fixture.clock.Set(fixture.clock.Now().Add(13 * time.Hour))
	expiredRequest := httptest.NewRequest(http.MethodGet, "/api/pos/ping", nil)
	expiredRequest.AddCookie(cookie)
	expiredResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(expiredResponse, expiredRequest)
	assertJSONError(t, expiredResponse, http.StatusUnauthorized, "unauthorized")
}
