package domain

// GroupModelsListConfig controls the optional custom /v1/models response list.
type GroupModelsListConfig struct {
	Enabled        bool     `json:"enabled"`
	Models         []string `json:"models,omitempty"`
	AllowFastMode  *bool    `json:"allow_fast_mode,omitempty"`
	AllowContext1M *bool    `json:"allow_context_1m,omitempty"`
}
