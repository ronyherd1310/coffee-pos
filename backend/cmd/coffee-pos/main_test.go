package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"coffee-pos/backend/internal/adapters/security"
)

func TestRunHashPINCommandPrintsBcryptHash(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"coffee-pos", "auth", "hash-pin", "123456"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success exit code, got %d with stderr %q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}

	hash := bytes.TrimSpace(stdout.Bytes())
	if len(hash) == 0 {
		t.Fatal("expected hash output")
	}

	matches, err := (security.BcryptPINHash{}).VerifyPINHash(context.Background(), "123456", string(hash))
	if err != nil {
		t.Fatalf("expected hash verification to succeed: %v", err)
	}
	if !matches {
		t.Fatal("expected printed hash to verify the input pin")
	}
}

func TestRunHashPINCommandRejectsInvalidPIN(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"coffee-pos", "auth", "hash-pin", "12345"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for invalid pin")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("expected stderr output for invalid pin")
	}
}

func TestRunUsageIncludesDatabaseCommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"coffee-pos"}, &stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("expected usage exit code, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "coffee-pos db migrate") {
		t.Fatalf("expected usage to include db migrate, got %q", stderr.String())
	}
}

func TestRunDBMigrateRejectsMissingDatabaseURL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"coffee-pos", "db", "migrate"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "DATABASE_URL is required") {
		t.Fatalf("expected missing database url error, got %q", stderr.String())
	}
}

func TestRunDBSeedRejectsMissingDatabaseURL(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"coffee-pos", "db", "seed"}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "DATABASE_URL is required") {
		t.Fatalf("expected missing database url error, got %q", stderr.String())
	}
}
