package setup

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestSetupPasswordPolicyMatchesAccountCreation(t *testing.T) {
	for _, password := range []string{"12345678", "12345678901", "123456789012", strings.Repeat("密", 12), strings.Repeat("a", 72), strings.Repeat("a", 73), strings.Repeat("密", 25)} {
		t.Run(password, func(t *testing.T) {
			setupErr := validatePassword(password)
			user := &service.User{}
			accountErr := user.SetPassword(password)
			if (setupErr == nil) != (accountErr == nil) {
				t.Fatalf("setup accepts password inconsistent with account creation: setup=%v account=%v", setupErr, accountErr)
			}
		})
	}
}
