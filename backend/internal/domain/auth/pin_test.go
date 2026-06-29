package auth

import "testing"

func TestValidatePINAcceptsExactlySixDigits(t *testing.T) {
	if err := ValidatePIN("123456"); err != nil {
		t.Fatalf("expected pin to be valid: %v", err)
	}
}

func TestValidatePINRejectsInvalidFormats(t *testing.T) {
	testCases := []string{
		"",
		"12345",
		"1234567",
		"12a456",
		" 123456",
		"123456 ",
		"１２３４５６",
	}

	for _, pin := range testCases {
		t.Run(pin, func(t *testing.T) {
			if err := ValidatePIN(pin); err == nil {
				t.Fatalf("expected pin %q to be rejected", pin)
			}
		})
	}
}
