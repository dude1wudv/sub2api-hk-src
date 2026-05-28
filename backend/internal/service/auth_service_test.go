package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsReservedEmail_DingTalkDomain(t *testing.T) {
	require.True(t, isReservedEmail("dingtalk-123@dingtalk-connect.invalid"))
	require.True(t, isReservedEmail("DINGTALK-456@DINGTALK-CONNECT.INVALID")) // case-insensitive
	require.False(t, isReservedEmail("real@dingtalk.com"))
}

func TestAuthServiceEnsureRegistrationCapacity(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	svc := &AuthService{entClient: client}

	for i := 0; i < maxRegisteredUsers-1; i++ {
		_, err := client.User.Create().
			SetEmail("capacity-ok-" + randomTestSuffix(t, i) + "@example.com").
			SetPasswordHash("hash").
			SetRole(RoleUser).
			SetStatus(StatusActive).
			Save(ctx)
		require.NoError(t, err)
	}
	require.NoError(t, svc.ensureRegistrationCapacity(ctx))

	_, err := client.User.Create().
		SetEmail("capacity-full@example.com").
		SetPasswordHash("hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		Save(ctx)
	require.NoError(t, err)

	err = svc.ensureRegistrationCapacity(ctx)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrRegistrationFull))
}

func randomTestSuffix(t *testing.T, i int) string {
	t.Helper()
	return strings.NewReplacer("/", "-", " ", "-").Replace(t.Name()) + "-" + strconv.Itoa(i)
}
