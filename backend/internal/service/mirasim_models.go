package service

// DefaultMirasimModelIDs is the conservative catalog enabled for new accounts.
func DefaultMirasimModelIDs() []string {
	return []string{
		"glm-5.3-flash", "kimi-k3",
	}
}

func DefaultMirasimModelMapping() map[string]any {
	mapping := make(map[string]any)
	for _, id := range DefaultMirasimModelIDs() {
		mapping[id] = id
	}
	return mapping
}
