package service

import "strings"

// DefaultMirasimModelIDs is the conservative catalog enabled for new accounts.
func DefaultMirasimModelIDs() []string {
	return []string{
		"glm-5.3-flash", "kimi-k3",
	}
}

// Native protocol selection is based on the mapped model, not the client URL.
func modelRoutedNativeProtocol(account *Account, model string) string {
	if account.Platform != PlatformMirasim {
		return openCodeGoNativeProtocol(account, model)
	}
	model = strings.TrimPrefix(model, "mirasim/")
	if strings.HasPrefix(model, "claude-") {
		return APIProtocolAnthropic
	}
	if strings.HasPrefix(model, "gpt-") {
		return APIProtocolResponses
	}
	return APIProtocolChatCompletions
}

func DefaultMirasimModelMapping() map[string]any {
	mapping := make(map[string]any)
	for _, id := range DefaultMirasimModelIDs() {
		mapping[id] = id
	}
	return mapping
}
