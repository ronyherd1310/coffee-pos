//go:build integration

package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthFlowOverHTTP(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	server := httptest.NewServer(fixture.router)
	defer server.Close()

	protectedBeforeLogin, err := http.Get(server.URL + "/api/pos/ping")
	if err != nil {
		t.Fatalf("expected protected request to succeed: %v", err)
	}
	assertHTTPError(t, protectedBeforeLogin, http.StatusUnauthorized, "unauthorized")

	loginResponse := mustDoJSONRequest(t, server.URL+"/api/auth/login", `{"pin":"123456"}`, nil)
	if loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected login success, got %d", loginResponse.StatusCode)
	}
	cookies := loginResponse.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one login cookie, got %d", len(cookies))
	}

	sessionResponse := mustDoRequest(t, http.MethodGet, server.URL+"/api/auth/session", nil, cookies)
	assertHTTPAuthenticated(t, sessionResponse, true)

	protectedAfterLogin := mustDoRequest(t, http.MethodGet, server.URL+"/api/pos/ping", nil, cookies)
	if protectedAfterLogin.StatusCode != http.StatusOK {
		t.Fatalf("expected protected request success, got %d", protectedAfterLogin.StatusCode)
	}

	logoutResponse := mustDoRequest(t, http.MethodPost, server.URL+"/api/auth/logout", nil, cookies)
	if logoutResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected logout success, got %d", logoutResponse.StatusCode)
	}
	logoutCookies := logoutResponse.Cookies()

	sessionAfterLogout := mustDoRequest(t, http.MethodGet, server.URL+"/api/auth/session", nil, logoutCookies)
	assertHTTPAuthenticated(t, sessionAfterLogout, false)
}

func TestRateLimitOverHTTP(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	server := httptest.NewServer(fixture.router)
	defer server.Close()

	for range 5 {
		response := mustDoJSONRequest(t, server.URL+"/api/auth/login", `{"pin":"000000"}`, nil)
		assertHTTPError(t, response, http.StatusUnauthorized, "invalid_pin")
	}

	response := mustDoJSONRequest(t, server.URL+"/api/auth/login", `{"pin":"123456"}`, nil)
	assertHTTPError(t, response, http.StatusTooManyRequests, "too_many_attempts")
}

func TestHealthRouteOverHTTP(t *testing.T) {
	server := httptest.NewServer(NewRouter())
	defer server.Close()

	response, err := http.Get(server.URL + "/api/health")
	if err != nil {
		t.Fatalf("expected health request to succeed: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.StatusCode)
	}
}

func mustDoJSONRequest(t *testing.T, url string, body string, cookies []*http.Cookie) *http.Response {
	t.Helper()

	return mustDoRequest(t, http.MethodPost, url, bytes.NewBufferString(body), cookies)
}

func mustDoRequest(t *testing.T, method string, url string, body *bytes.Buffer, cookies []*http.Cookie) *http.Response {
	t.Helper()

	var reader *bytes.Buffer
	if body == nil {
		reader = bytes.NewBuffer(nil)
	} else {
		reader = body
	}

	request, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("expected request creation to succeed: %v", err)
	}
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("expected HTTP request to succeed: %v", err)
	}

	return response
}

func assertHTTPError(t *testing.T, response *http.Response, expectedStatus int, expectedError string) {
	t.Helper()
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, response.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if body["error"] != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, body["error"])
	}
}

func assertHTTPAuthenticated(t *testing.T, response *http.Response, expected bool) {
	t.Helper()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	var body struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON session response: %v", err)
	}
	if body.Authenticated != expected {
		t.Fatalf("expected authenticated=%t, got %t", expected, body.Authenticated)
	}
}
