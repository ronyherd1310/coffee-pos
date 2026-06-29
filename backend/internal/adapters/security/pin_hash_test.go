package security

import (
	"context"
	"testing"
)

func TestBcryptPINHashMatchesCorrectPIN(t *testing.T) {
	hasher := BcryptPINHash{}

	hash, err := hasher.HashPIN("123456")
	if err != nil {
		t.Fatalf("expected hash generation to succeed: %v", err)
	}

	matches, err := hasher.VerifyPINHash(context.Background(), "123456", hash)
	if err != nil {
		t.Fatalf("expected hash verification to succeed: %v", err)
	}
	if !matches {
		t.Fatal("expected correct pin to match hash")
	}
}

func TestBcryptPINHashRejectsWrongPIN(t *testing.T) {
	hasher := BcryptPINHash{}

	hash, err := hasher.HashPIN("123456")
	if err != nil {
		t.Fatalf("expected hash generation to succeed: %v", err)
	}

	matches, err := hasher.VerifyPINHash(context.Background(), "654321", hash)
	if err != nil {
		t.Fatalf("expected wrong pin verification to return false without error: %v", err)
	}
	if matches {
		t.Fatal("expected wrong pin not to match hash")
	}
}

func TestBcryptPINHashValidatesPINFormatBeforeHashing(t *testing.T) {
	hasher := BcryptPINHash{}

	_, err := hasher.HashPIN("12345")
	if err == nil {
		t.Fatal("expected invalid pin format to fail")
	}
}
