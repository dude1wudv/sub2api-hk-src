package service

// DefaultMirasimModelIDs is the model catalog exposed by Mirasim's relay.
func DefaultMirasimModelIDs() []string {
	return []string{
		"claude-opus-5", "claude-sonnet-5", "claude-fable-5", "claude-haiku-4-5",
		"claude-opus-4-8", "gpt-6-astra", "kimi-k3", "deepseek-flash",
		"deepseek-v4-flash", "glm-5.3-flash",
	}
}

func DefaultMirasimModelMapping() map[string]any {
	mapping := make(map[string]any)
	for _, id := range DefaultMirasimModelIDs() {
		mapping[id] = id
	}
	return mapping
}
