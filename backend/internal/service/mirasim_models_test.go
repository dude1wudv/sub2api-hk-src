package service

import "testing"

func TestMirasimDefaultModelsRestrictImportedAndLegacyAccounts(t *testing.T) {
	models := DefaultMirasimModelIDs()
	if len(models) == 0 {
		t.Fatal("Mirasim model catalog is empty")
	}
	for _, credentials := range []map[string]any{
		{},
		{"model_mapping": DefaultMirasimModelMapping()},
	} {
		account := &Account{Platform: PlatformMirasim, Type: AccountTypeAPIKey, Credentials: credentials}
		for _, model := range models {
			if !account.IsModelSupported(model) {
				t.Errorf("Mirasim account should support %s", model)
			}
		}
		if account.IsModelSupported("unsupported-model") {
			t.Error("Mirasim account accepted a model outside its catalog")
		}
	}
}
