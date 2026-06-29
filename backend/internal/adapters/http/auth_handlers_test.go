package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	appauth "coffee-pos/backend/internal/app/auth"
	"coffee-pos/backend/internal/adapters/security"
)

func TestLoginReturnsInvalidPINForMalformedRequests(t *testing.T) {
	fixture := newAuthRouterFixture(t)

	testCases := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "invalid json", body: `{"pin":`},
		{name: "missing pin", body: `{"other":"123456"}`},
		{name: "null pin", body: `{"pin":null}`},
		{name: "numeric pin", body: `{"pin":123456}`},
		{name: "short pin", body: `{"pin":"12345"}`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(testCase.body))
			request.RemoteAddr = "203.0.113.10:1234"
			response := httptest.NewRecorder()

			fixture.router.ServeHTTP(response, request)

			assertJSONError(t, response, http.StatusUnauthorized, "invalid_pin")
			if cookies := response.Result().Cookies(); len(cookies) != 0 {
				t.Fatalf("expected no cookie to be set, got %d", len(cookies))
			}
		})
	}
}

func TestLoginRejectsOversizedBodies(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	body := `{"pin":"123456","padding":"` + strings.Repeat("x", 4096) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	request.RemoteAddr = "203.0.113.10:1234"
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusUnauthorized, "invalid_pin")
}

func TestLoginIgnoresUnexpectedExtraJSONFields(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"pin":"123456","extra":"ignored"}`))
	request.RemoteAddr = "203.0.113.10:1234"
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d", response.Code)
	}
}

func TestLoginReturnsTooManyAttemptsAndSkipsVerificationWhenBlocked(t *testing.T) {
	fixture := newAuthRouterFixture(t)

	for range 5 {
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"pin":"000000"}`))
		request.RemoteAddr = "203.0.113.10:1234"
		response := httptest.NewRecorder()

		fixture.router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected invalid pin before rate limiting, got %d", response.Code)
		}
	}

	callsBeforeBlockedRequest := fixture.verifier.calls
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"pin":"123456"}`))
	request.RemoteAddr = "203.0.113.10:1234"
	response := httptest.NewRecorder()

	fixture.router.ServeHTTP(response, request)

	assertJSONError(t, response, http.StatusTooManyRequests, "too_many_attempts")
	if fixture.verifier.calls != callsBeforeBlockedRequest {
		t.Fatalf("expected blocked request to skip hash verification, got %d calls before and %d after", callsBeforeBlockedRequest, fixture.verifier.calls)
	}
}

func TestLoginSetsSessionCookieAndSessionEndpointUsesIt(t *testing.T) {
	fixture := newAuthRouterFixture(t)

	loginResponse := fixture.login(t, `{"pin":"123456"}`, nil, "203.0.113.10:1234")
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d", loginResponse.Code)
	}

	cookie := assertSingleCookie(t, loginResponse, fixture.cookieName)
	if !cookie.HttpOnly {
		t.Fatal("expected session cookie to be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}
	if !cookie.Secure {
		t.Fatal("expected secure cookie when configured")
	}
	if cookie.Path != "/" {
		t.Fatalf("expected cookie path /, got %q", cookie.Path)
	}
	expectedExpiry := fixture.clock.Now().Add(12 * time.Hour)
	if !cookie.Expires.Equal(expectedExpiry) {
		t.Fatalf("expected cookie expiry %s, got %s", expectedExpiry, cookie.Expires)
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionRequest.AddCookie(cookie)
	sessionResponse := httptest.NewRecorder()

	fixture.router.ServeHTTP(sessionResponse, sessionRequest)

	assertSessionAuthenticated(t, sessionResponse, true)
}

func TestRepeatedLoginReplacesOnlyCurrentSession(t *testing.T) {
	fixture := newAuthRouterFixture(t)

	firstLogin := fixture.login(t, `{"pin":"123456"}`, nil, "203.0.113.10:1111")
	firstCookie := assertSingleCookie(t, firstLogin, fixture.cookieName)

	secondClientLogin := fixture.login(t, `{"pin":"123456"}`, nil, "198.51.100.2:2222")
	secondCookie := assertSingleCookie(t, secondClientLogin, fixture.cookieName)

	repeatLogin := fixture.login(t, `{"pin":"123456"}`, []*http.Cookie{firstCookie}, "203.0.113.10:1111")
	freshCookie := assertSingleCookie(t, repeatLogin, fixture.cookieName)
	if freshCookie.Value == firstCookie.Value {
		t.Fatal("expected repeated login to issue a fresh session id")
	}

	oldSessionRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	oldSessionRequest.AddCookie(firstCookie)
	oldSessionResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(oldSessionResponse, oldSessionRequest)
	assertSessionAuthenticated(t, oldSessionResponse, false)

	otherSessionRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	otherSessionRequest.AddCookie(secondCookie)
	otherSessionResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(otherSessionResponse, otherSessionRequest)
	assertSessionAuthenticated(t, otherSessionResponse, true)
}

func TestLogoutClearsCookieAndIsIdempotent(t *testing.T) {
	fixture := newAuthRouterFixture(t)
	loginResponse := fixture.login(t, `{"pin":"123456"}`, nil, "203.0.113.10:1234")
	cookie := assertSingleCookie(t, loginResponse, fixture.cookieName)

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutRequest.AddCookie(cookie)
	logoutResponse := httptest.NewRecorder()

	fixture.router.ServeHTTP(logoutResponse, logoutRequest)

	if logoutResponse.Code != http.StatusOK {
		t.Fatalf("expected logout success, got %d", logoutResponse.Code)
	}
	cleared := assertSingleCookie(t, logoutResponse, fixture.cookieName)
	if cleared.Value != "" {
		t.Fatalf("expected cleared cookie value to be empty, got %q", cleared.Value)
	}
	if cleared.MaxAge != -1 {
		t.Fatalf("expected cleared cookie MaxAge -1, got %d", cleared.MaxAge)
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionRequest.AddCookie(cookie)
	sessionResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(sessionResponse, sessionRequest)
	assertSessionAuthenticated(t, sessionResponse, false)

	secondLogoutRequest := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	secondLogoutRequest.AddCookie(cookie)
	secondLogoutResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(secondLogoutResponse, secondLogoutRequest)
	if secondLogoutResponse.Code != http.StatusOK {
		t.Fatalf("expected repeated logout success, got %d", secondLogoutResponse.Code)
	}
}

func TestSessionEndpointReturnsFalseForUnknownAndExpiredCookies(t *testing.T) {
	fixture := newAuthRouterFixture(t)

	unknownRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	unknownRequest.AddCookie(&http.Cookie{Name: fixture.cookieName, Value: "unknown"})
	unknownResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(unknownResponse, unknownRequest)
	assertSessionAuthenticated(t, unknownResponse, false)

	loginResponse := fixture.login(t, `{"pin":"123456"}`, nil, "203.0.113.10:1234")
	cookie := assertSingleCookie(t, loginResponse, fixture.cookieName)

	fixture.clock.Set(fixture.clock.Now().Add(13 * time.Hour))
	expiredRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	expiredRequest.AddCookie(cookie)
	expiredResponse := httptest.NewRecorder()
	fixture.router.ServeHTTP(expiredResponse, expiredRequest)
	assertSessionAuthenticated(t, expiredResponse, false)
}

type authRouterFixture struct {
	router     http.Handler
	clock      *mutableClock
	verifier   *countingVerifier
	cookieName string
}

func newAuthRouterFixture(t *testing.T) authRouterFixture {
	t.Helper()

	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("expected Asia/Jakarta location to load: %v", err)
	}

	now := time.Date(2026, 6, 29, 10, 0, 0, 0, jakarta)
	hasher := security.BcryptPINHash{}
	hash, err := hasher.HashPIN("123456")
	if err != nil {
		t.Fatalf("expected hash generation to succeed: %v", err)
	}

	verifier := &countingVerifier{delegate: hasher}
	clock := &mutableClock{now: now}
	service := appauth.NewService(appauth.Dependencies{
		CashierPINHash: hash,
		Verifier:       verifier,
		Sessions:       security.NewInMemorySessionStore(),
		RateLimiter:    security.NewInMemoryRateLimiter(),
		SessionIDs:     &sequentialSessionIDGenerator{},
		Clock:          clock,
		Location:       jakarta,
	})

	cookieName := "coffee_pos_session"
	return authRouterFixture{
		router: NewRouter(RouterOptions{
			AuthService: service,
			Cookie: CookieConfig{
				Name:     cookieName,
				Path:     "/",
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			},
		}),
		clock:      clock,
		verifier:   verifier,
		cookieName: cookieName,
	}
}

func (f authRouterFixture) login(t *testing.T, body string, cookies []*http.Cookie, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	request.RemoteAddr = remoteAddr
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()

	f.router.ServeHTTP(response, request)

	return response
}

func assertJSONError(t *testing.T, response *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if response.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, response.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(bytes.NewReader(response.Body.Bytes())).Decode(&body); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if body["error"] != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, body["error"])
	}
}

func assertSessionAuthenticated(t *testing.T, response *httptest.ResponseRecorder, expected bool) {
	t.Helper()

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var body struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(bytes.NewReader(response.Body.Bytes())).Decode(&body); err != nil {
		t.Fatalf("expected JSON session response: %v", err)
	}
	if body.Authenticated != expected {
		t.Fatalf("expected authenticated=%t, got %t", expected, body.Authenticated)
	}
}

func assertSingleCookie(t *testing.T, response *httptest.ResponseRecorder, expectedName string) *http.Cookie {
	t.Helper()

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if cookies[0].Name != expectedName {
		t.Fatalf("expected cookie %q, got %q", expectedName, cookies[0].Name)
	}
	return cookies[0]
}

type mutableClock struct {
	now time.Time
}

func (c *mutableClock) Now() time.Time {
	return c.now
}

func (c *mutableClock) Set(now time.Time) {
	c.now = now
}

type countingVerifier struct {
	delegate security.BcryptPINHash
	calls    int
}

func (v *countingVerifier) VerifyPINHash(ctx context.Context, pin string, hash string) (bool, error) {
	v.calls++
	return v.delegate.VerifyPINHash(ctx, pin, hash)
}

type sequentialSessionIDGenerator struct {
	next int
}

func (g *sequentialSessionIDGenerator) NewSessionID() (string, error) {
	g.next++
	return "session-" + strconv.Itoa(g.next), nil
}
