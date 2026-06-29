package security

import (
	"encoding/hex"
	"testing"
)

func TestRandomSessionIDGeneratorReturnsHexEncodedRandomIDs(t *testing.T) {
	generator := RandomSessionIDGenerator{}

	first, err := generator.NewSessionID()
	if err != nil {
		t.Fatalf("expected first session id generation to succeed: %v", err)
	}
	second, err := generator.NewSessionID()
	if err != nil {
		t.Fatalf("expected second session id generation to succeed: %v", err)
	}

	if len(first) != 64 {
		t.Fatalf("expected 64-character session id, got %d", len(first))
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Fatalf("expected hex session id, got %q: %v", first, err)
	}
	if first == second {
		t.Fatal("expected generated session ids to differ")
	}
}
