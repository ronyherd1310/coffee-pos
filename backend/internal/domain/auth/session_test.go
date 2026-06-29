package auth

import (
	"testing"
	"time"
)

func TestSessionExpiryUsesEarlierOfTwelveHoursOrJakartaMidnight(t *testing.T) {
	jakarta := mustJakartaLocation(t)

	loginAt := time.Date(2026, 6, 29, 10, 0, 0, 0, jakarta)
	expiresAt := SessionExpiry(loginAt, jakarta)

	expected := time.Date(2026, 6, 29, 22, 0, 0, 0, jakarta)
	if !expiresAt.Equal(expected) {
		t.Fatalf("expected expiry %s, got %s", expected, expiresAt)
	}
}

func TestSessionExpiryUsesEndOfDayWhenItComesFirst(t *testing.T) {
	jakarta := mustJakartaLocation(t)

	loginAt := time.Date(2026, 6, 29, 18, 0, 0, 0, jakarta)
	expiresAt := SessionExpiry(loginAt, jakarta)

	expected := time.Date(2026, 6, 30, 0, 0, 0, 0, jakarta)
	if !expiresAt.Equal(expected) {
		t.Fatalf("expected expiry %s, got %s", expected, expiresAt)
	}
}

func TestSessionExpiryHandlesBoundaryTimes(t *testing.T) {
	jakarta := mustJakartaLocation(t)

	testCases := []struct {
		name      string
		loginAt   time.Time
		expiresAt time.Time
	}{
		{
			name:      "just before midnight",
			loginAt:   time.Date(2026, 6, 29, 23, 59, 0, 0, jakarta),
			expiresAt: time.Date(2026, 6, 30, 0, 0, 0, 0, jakarta),
		},
		{
			name:      "exactly midnight",
			loginAt:   time.Date(2026, 6, 29, 0, 0, 0, 0, jakarta),
			expiresAt: time.Date(2026, 6, 29, 12, 0, 0, 0, jakarta),
		},
		{
			name:      "exactly twelve hours before midnight",
			loginAt:   time.Date(2026, 6, 29, 12, 0, 0, 0, jakarta),
			expiresAt: time.Date(2026, 6, 30, 0, 0, 0, 0, jakarta),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			expiresAt := SessionExpiry(testCase.loginAt, jakarta)

			if !expiresAt.Equal(testCase.expiresAt) {
				t.Fatalf("expected expiry %s, got %s", testCase.expiresAt, expiresAt)
			}
			if !expiresAt.After(testCase.loginAt) {
				t.Fatalf("expected expiry %s to be after login %s", expiresAt, testCase.loginAt)
			}
		})
	}
}

func mustJakartaLocation(t *testing.T) *time.Location {
	t.Helper()

	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("expected Asia/Jakarta location to load: %v", err)
	}

	return location
}
