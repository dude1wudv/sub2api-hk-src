package service

// DefaultMirasimModelIDs is the model catalog returned by Mirasim's relay.
func DefaultMirasimModelIDs() []string {
	return []string{
		"claude-opus-5-5", "deepseek-flash", "deepseek-v4-flash",
		"deepseek-v4-flash-vision-exp", "glm-5.3-flash", "gpt-6-luna",
		"gpt-6-sol", "kimi-k3",
	}
}

func DefaultMirasimModelMapping() map[string]any {
	mapping := make(map[string]any)
	for _, id := range DefaultMirasimModelIDs() {
		mapping[id] = id
	}
	return mapping
}
