package auth

import "errors"

var ErrInvalidPINFormat = errors.New("invalid pin format")

func ValidatePIN(pin string) error {
	if len(pin) != 6 {
		return ErrInvalidPINFormat
	}

	for _, character := range pin {
		if character < '0' || character > '9' {
			return ErrInvalidPINFormat
		}
	}

	return nil
}
