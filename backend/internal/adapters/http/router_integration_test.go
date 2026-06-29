//go:build integration

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
