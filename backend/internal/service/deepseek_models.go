package service

var defaultDeepSeekModelIDs = []string{
	"deepseek-chat",
	"deepseek-reasoner",
}

func deepSeekDefaultModelIDs() []string {
	ids := make([]string, len(defaultDeepSeekModelIDs))
	copy(ids, defaultDeepSeekModelIDs)
	return ids
}
