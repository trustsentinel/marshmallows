package auth

import (
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// generateTOTP creates a new TOTP secret bound to an account for display as a
// provisioning URI / QR code.
func generateTOTP(account string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{Issuer: "marshmallows", AccountName: account})
}

func validateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}
