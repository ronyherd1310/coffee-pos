package config

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestLoadRejectsMissingCashierPINHash(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("APP_ENV", "development")

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing cashier pin hash to fail")
	}
}

func TestLoadRejectsMalformedCashierPINHash(t *testing.T) {
	t.Setenv("CASHIER_PIN_HASH", "not-a-valid-hash")

	_, err := Load()
	if err == nil {
		t.Fatal("expected malformed cashier pin hash to fail")
	}
}

func TestLoadForcesSecureCookiesInProduction(t *testing.T) {
	t.Setenv("CASHIER_PIN_HASH", mustHashPIN(t, "123456"))
	t.Setenv("APP_ENV", "production")
	t.Setenv("SESSION_COOKIE_SECURE", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}

	if !cfg.SessionCookieSecure {
		t.Fatal("expected production config to force secure cookies")
	}
}

func TestLoadRejectsInvalidSessionCookieSecureValue(t *testing.T) {
	t.Setenv("CASHIER_PIN_HASH", mustHashPIN(t, "123456"))
	t.Setenv("SESSION_COOKIE_SECURE", "maybe")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid secure cookie value to fail")
	}
}

func TestLoadRejectsEmptySessionCookieName(t *testing.T) {
	t.Setenv("CASHIER_PIN_HASH", mustHashPIN(t, "123456"))
	t.Setenv("SESSION_COOKIE_NAME", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected empty cookie name to fail")
	}
}

func TestLoadSetsJakartaLocation(t *testing.T) {
	t.Setenv("CASHIER_PIN_HASH", mustHashPIN(t, "123456"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}

	if cfg.BusinessLocation.String() != "Asia/Jakarta" {
		t.Fatalf("expected Asia/Jakarta location, got %q", cfg.BusinessLocation.String())
	}
}

func TestLoadDatabaseRejectsMissingDatabaseURL(t *testing.T) {
	_, err := LoadDatabase()
	if err == nil {
		t.Fatal("expected missing database url to fail")
	}
}

func TestLoadDatabaseUsesDatabaseURLAndConservativePoolDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://coffee_pos:coffee_pos_dev@localhost:5432/coffee_pos?sslmode=disable")

	cfg, err := LoadDatabase()
	if err != nil {
		t.Fatalf("expected database config to load: %v", err)
	}

	if cfg.URL != "postgres://coffee_pos:coffee_pos_dev@localhost:5432/coffee_pos?sslmode=disable" {
		t.Fatalf("expected database url from env, got %q", cfg.URL)
	}
	if cfg.MaxOpenConns != 3 {
		t.Fatalf("expected max open conns 3, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 1 {
		t.Fatalf("expected max idle conns 1, got %d", cfg.MaxIdleConns)
	}
}

func mustHashPIN(t *testing.T, pin string) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("expected bcrypt hash generation to succeed: %v", err)
	}

	return string(hash)
}
