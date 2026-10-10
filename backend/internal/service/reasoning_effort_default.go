package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ReasoningEffortDefaultConfig controls protocol-level reasoning injection when
// the client did not provide an explicit effort.
type ReasoningEffortDefaultConfig struct {
	Enabled bool                         `json:"enabled"`
	Rules   []ReasoningEffortDefaultRule `json:"rules"`
}

type ReasoningEffortDefaultRule struct {
	Platform  string `json:"platform"`
	MatchType string `json:"match_type"`
	Model     string `json:"model"`
	Effort    string `json:"effort"`
}

const (
	ReasoningEffortSourceExplicit    = "explicit"
	ReasoningEffortSourceDefault     = "default"
	ReasoningEffortSourceModelSuffix = "model_suffix"
)

func NormalizeReasoningEffortDefaultConfig(cfg ReasoningEffortDefaultConfig) (ReasoningEffortDefaultConfig, error) {
	if len(cfg.Rules) > maxReasoningEffortMappings {
		return ReasoningEffortDefaultConfig{}, fmt.Errorf("too many reasoning default rules")
	}
	normalized := ReasoningEffortDefaultConfig{Enabled: cfg.Enabled, Rules: make([]ReasoningEffortDefaultRule, 0, len(cfg.Rules))}
	seen := make(map[string]struct{}, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		platform := strings.ToLower(strings.TrimSpace(rule.Platform))
		if platform != PlatformOpenAI && platform != PlatformAnthropic && platform != PlatformComposite {
			return ReasoningEffortDefaultConfig{}, fmt.Errorf("unsupported reasoning default platform %q", rule.Platform)
		}
		matchType, err := normalizeReasoningEffortMatchType(rule.MatchType, rule.Model)
		if err != nil {
			return ReasoningEffortDefaultConfig{}, err
		}
		model := strings.TrimSpace(rule.Model)
		if len(model) > maxReasoningEffortModelLen {
			return ReasoningEffortDefaultConfig{}, fmt.Errorf("reasoning default model is too long")
		}
		effort, err := normalizeMaxReasoningEffortForPlatform(platform, rule.Effort)
		if err != nil || effort == "" {
			return ReasoningEffortDefaultConfig{}, fmt.Errorf("unsupported reasoning default effort %q for %s", rule.Effort, platform)
		}
		if matchType == "exact" && !effortAllowedForDefault(platform, model, effort) {
			return ReasoningEffortDefaultConfig{}, fmt.Errorf("unsupported reasoning default effort %q for %s/%s", rule.Effort, platform, model)
		}
		if openai.IsGPT61SolModelSpelling(model) && (effort == "minimal" || effort == "none") {
			return ReasoningEffortDefaultConfig{}, fmt.Errorf("GPT-6.1 Sol does not support %s", effort)
		}
		key := strings.Join([]string{platform, matchType, strings.ToLower(model)}, "\x00")
		if _, ok := seen[key]; ok {
			return ReasoningEffortDefaultConfig{}, fmt.Errorf("duplicate reasoning default rule for %s", model)
		}
		seen[key] = struct{}{}
		normalized.Rules = append(normalized.Rules, ReasoningEffortDefaultRule{Platform: platform, MatchType: matchType, Model: model, Effort: effort})
	}
	return normalized, nil
}

func ParseReasoningEffortDefaultConfig(raw string) (ReasoningEffortDefaultConfig, error) {
	if strings.TrimSpace(raw) == "" {
		return ReasoningEffortDefaultConfig{Rules: []ReasoningEffortDefaultRule{}}, nil
	}
	var cfg ReasoningEffortDefaultConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return ReasoningEffortDefaultConfig{}, fmt.Errorf("invalid reasoning default config: %w", err)
	}
	return NormalizeReasoningEffortDefaultConfig(cfg)
}

func effortAllowedForDefault(platform, model, effort string) bool {
	if platform == PlatformComposite {
		return effortAllowedForDefault(PlatformOpenAI, model, effort) || effortAllowedForDefault(PlatformAnthropic, model, effort)
	}
	if platform == PlatformAnthropic {
		for _, value := range claude.EffortLevelsForModel(model) {
			if value == effort {
				return true
			}
		}
		return false
	}
	if platform != PlatformOpenAI || !openai.IsGPT61SolModelSpelling(model) {
		return false
	}
	if rank, ok := reasoningEffortRank(effort); ok {
		return rank > 0 && effort != "none" && effort != "minimal"
	}
	return false
}

func explicitReasoningEffort(body []byte, protocol string, model string) (string, string, error) {
	paths := []string{"reasoning.effort", "reasoning_effort", "output_config.effort"}
	if protocol == "messages" {
		paths = []string{"output_config.effort", "thinking.type"}
	}
	var found string
	for _, path := range paths {
		value := gjson.GetBytes(body, path)
		if !value.Exists() || value.Type == gjson.Null {
			continue
		}
		if value.Type != gjson.String {
			return "", "", fmt.Errorf("%s must be a string or null", path)
		}
		raw := strings.TrimSpace(value.String())
		if path == "thinking.type" {
			if strings.EqualFold(raw, "disabled") {
				raw = "none"
			} else {
				continue
			}
		}
		if raw == "" {
			continue
		}
		canonical := normalizeReasoningEffortMappingSource(raw)
		if canonical == "" {
			// Unknown provider-specific explicit values remain upstream-owned.
			// Only injected defaults are constrained to the capability catalog.
			canonical = strings.ToLower(raw)
		}
		if found != "" && found != canonical {
			return "", "", fmt.Errorf("conflicting reasoning effort values")
		}
		found = canonical
	}
	if found != "" {
		return found, ReasoningEffortSourceExplicit, nil
	}
	if suffix := canonicalReasoningEffortFromModelSuffix(model); suffix != "" {
		return suffix, ReasoningEffortSourceModelSuffix, nil
	}
	return "", "", nil
}

// ResolveReasoningEffortDefault returns a copied body with a protocol-native
// default only when no explicit effort was supplied and a capability-checked
// rule matches. It never creates a Messages thinking block.
func ResolveReasoningEffortDefault(body []byte, inboundProtocol, platform, model string, cfg ReasoningEffortDefaultConfig) ([]byte, string, error) {
	if len(body) == 0 {
		return body, "", nil
	}
	model = strings.TrimSpace(model)
	_, source, err := explicitReasoningEffort(body, inboundProtocol, model)
	if err != nil {
		return body, "", err
	}
	if source == ReasoningEffortSourceExplicit || source == ReasoningEffortSourceModelSuffix || !cfg.Enabled {
		return body, source, nil
	}
	mappings := make([]ReasoningEffortMapping, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		if rule.Platform == platform || rule.Platform == PlatformComposite {
			mappings = append(mappings, ReasoningEffortMapping{
				From: "high", To: rule.Effort, MatchType: rule.MatchType, Model: rule.Model,
			})
		}
	}
	mapping, matched := selectReasoningEffortMapping(mappings, "high", model)
	if !matched {
		return body, "", nil
	}
	effort := mapping.To
	if !effortAllowedForDefault(platform, model, effort) {
		slog.Debug("gateway_reasoning_default_skipped", "platform", platform, "model", model, "effort", effort)
		return body, "", nil
	}
	path := "reasoning.effort"
	switch inboundProtocol {
	case "chat":
		path = "reasoning_effort"
	case "messages":
		path = "output_config.effort"
	}
	updated, err := sjson.SetBytes(body, path, effort)
	if err != nil {
		return body, "", fmt.Errorf("inject reasoning default: %w", err)
	}
	return updated, ReasoningEffortSourceDefault, nil
}
