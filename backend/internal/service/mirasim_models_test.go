package service

import "testing"

func TestMirasimNativeProtocolByModel(t *testing.T) {
	account := &Account{Platform: PlatformMirasim, Type: AccountTypeAPIKey}
	tests := []struct {
		model string
		want  string
	}{
		{model: "deepseek-v4.1-flash", want: APIProtocolChatCompletions},
		{model: "mirasim/glm-5.3-flash", want: APIProtocolChatCompletions},
		{model: "kimi-k3", want: APIProtocolChatCompletions},
		{model: "claude-opus-5-5", want: APIProtocolAnthropic},
		{model: "gpt-6-astra", want: APIProtocolResponses},
	}
	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := modelRoutedNativeProtocol(account, tt.model); got != tt.want {
				t.Fatalf("modelRoutedNativeProtocol(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestDefaultMirasimModelIDs(t *testing.T) {
	want := []string{"deepseek-v4.1-flash", "glm-5.3-flash", "kimi-k3"}
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
		for _, model := range []string{"claude-opus-5-5", "deepseek-v4-flash", "gpt-6-sol", "unsupported-model"} {
			if account.IsModelSupported(model) {
				t.Errorf("Mirasim account accepted model outside the default catalog: %s", model)
			}
		}
	}
}
