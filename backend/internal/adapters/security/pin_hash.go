package security

import (
	"context"

	authdomain "coffee-pos/backend/internal/domain/auth"
	"golang.org/x/crypto/bcrypt"
)

type BcryptPINHash struct {
	Cost int
}

func (b BcryptPINHash) VerifyPINHash(_ context.Context, pin string, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin))
	if err == nil {
		return true, nil
	}
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return false, nil
	}
	return false, err
}

func (b BcryptPINHash) HashPIN(pin string) (string, error) {
	if err := authdomain.ValidatePIN(pin); err != nil {
		return "", err
	}

	cost := b.Cost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pin), cost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}
