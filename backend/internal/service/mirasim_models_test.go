package service

import "testing"

func TestMirasimIsNotRegisteredForModelRouting(t *testing.T) {
	if IsMultiProtocolAPIKeyProvider(PlatformMirasim) {
		t.Fatal("retired Mirasim platform must not participate in model routing")
	}
	if _, ok := DetectModelPlatform("mirasim/glm-5.3-flash"); ok {
		t.Fatal("retired Mirasim model prefix must not create a composite route")
	}
}

func TestDefaultMirasimModelIDs(t *testing.T) {
	want := []string{"glm-5.3-flash", "kimi-k3"}
	got := DefaultMirasimModelIDs()
	if len(got) != len(want) {
		t.Fatalf("DefaultMirasimModelIDs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DefaultMirasimModelIDs() = %v, want %v", got, want)
		}
	}
}

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
		for _, model := range []string{"claude-opus-5-5", "deepseek-v4.1-flash", "deepseek-v4-flash", "gpt-6-sol", "unsupported-model"} {
			if account.IsModelSupported(model) {
				t.Errorf("Mirasim account accepted model outside the default catalog: %s", model)
			}
		}
	}
}
