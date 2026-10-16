//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type smartRoutingRetirementAccountRepo struct {
	AccountRepository
	accounts []Account
}

func (r *smartRoutingRetirementAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return r.accounts, nil
}

func TestOpenAISmartRoutingModelsRespectsRetirementAfterPublicAliasProjection(t *testing.T) {
	account := func(id int64, typ string, mapping map[string]any) Account {
		return Account{
			ID: id, Platform: PlatformOpenAI, Type: typ,
			Status: StatusActive, Schedulable: true,
			Credentials: map[string]any{"model_mapping": mapping},
		}
	}
	retiredAlias := func(id int64, typ string) Account {
		return account(id, typ, map[string]any{"legacy-alias": "gpt-5.5", "live-alias": "gpt-6.1-sol"})
	}
	for _, tc := range []struct {
		name     string
		accounts []Account
		want     []string
	}{
		{
			name:     "OAuth alias to retired target is hidden",
			accounts: []Account{retiredAlias(1, AccountTypeOAuth)},
			want:     []string{"live-alias"},
		},
		{
			name:     "setup token follows subscription retirement",
			accounts: []Account{retiredAlias(1, AccountTypeSetupToken)},
			want:     []string{"live-alias"},
		},
		{
			name:     "mixed API key preserves servable alias",
			accounts: []Account{retiredAlias(1, AccountTypeOAuth), retiredAlias(2, AccountTypeAPIKey)},
			want:     []string{"legacy-alias", "live-alias"},
		},
		{
			name: "unrelated API key does not resurrect retired alias",
			accounts: []Account{
				retiredAlias(1, AccountTypeOAuth),
				account(2, AccountTypeAPIKey, map[string]any{"live-alias": "gpt-6.1-sol"}),
			},
			want: []string{"live-alias"},
		},
		{
			name: "old public name mapped to replacement is retained",
			accounts: []Account{account(1, AccountTypeOAuth, map[string]any{
				"gpt-5.5": "gpt-6.1-sol", "legacy-alias": "gpt-5.5",
			})},
			want: []string{"gpt-5.5"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			groupID := int64(74)
			svc := &OpenAIGatewayService{accountRepo: &smartRoutingRetirementAccountRepo{accounts: tc.accounts}}
			got, err := svc.SmartRoutingModels(context.Background(), &APIKey{
				GroupID: &groupID,
				Group:   &Group{ID: groupID, Platform: PlatformOpenAI},
			})
			require.NoError(t, err)
			require.ElementsMatch(t, tc.want, got)
		})
	}
}
