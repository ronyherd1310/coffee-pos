package security

import (
	"crypto/rand"
	"encoding/hex"
)

const sessionIDBytes = 32

type RandomSessionIDGenerator struct{}

func (RandomSessionIDGenerator) NewSessionID() (string, error) {
	raw := make([]byte, sessionIDBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return hex.EncodeToString(raw), nil
}
