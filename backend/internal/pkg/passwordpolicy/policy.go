// Package passwordpolicy defines the policy for newly chosen passwords only.
package passwordpolicy

import (
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const MinLength = 12
const MaxBytes = 72 // bcrypt rejects larger inputs; never truncate a password.

type Policy struct {
	MinLength int `json:"min_length"`
	MaxBytes  int `json:"max_bytes"`
}

func Public() Policy { return Policy{MinLength: MinLength, MaxBytes: MaxBytes} }

// Validate deliberately does not trim, normalize, or require character classes.
// Authentication of existing passwords must not call this function.
func Validate(password string) error {
	if !utf8.ValidString(password) {
		return infraerrors.BadRequest("PASSWORD_INVALID", "password must contain valid Unicode characters")
	}
	if utf8.RuneCountInString(password) < MinLength {
		return infraerrors.BadRequest("PASSWORD_TOO_SHORT", "password must contain at least 12 characters")
	}
	if len(password) > MaxBytes {
		return infraerrors.BadRequest("PASSWORD_TOO_LONG", "password must not exceed 72 UTF-8 bytes")
	}
	return nil
}
