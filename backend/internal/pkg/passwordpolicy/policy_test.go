package passwordpolicy

import (
	"strings"
	"testing"
)

func TestNewPasswordPolicy(t *testing.T) {
	for _, tt := range []struct {
		name, password string
		valid          bool
	}{
		{"short", "password123", false},
		{"minimum", "password1234", true},
		{"unicode characters", strings.Repeat("密", 12), true},
		{"unicode too few", strings.Repeat("密", 11), false},
		{"bcrypt boundary", strings.Repeat("a", 72), true},
		{"bcrypt overflow", strings.Repeat("a", 73), false},
		{"unicode byte overflow", strings.Repeat("密", 25), false},
		{"spaces retained", " twelve chars ", true},
		{"invalid UTF8", strings.Repeat("a", 12) + "\xff", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if (Validate(tt.password) == nil) != tt.valid {
				t.Fatalf("unexpected validation result")
			}
		})
	}
}
