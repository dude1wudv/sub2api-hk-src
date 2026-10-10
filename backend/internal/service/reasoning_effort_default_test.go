package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func defaultReasoningTestConfig(platform, model, effort string) ReasoningEffortDefaultConfig {
	return ReasoningEffortDefaultConfig{Enabled: true, Rules: []ReasoningEffortDefaultRule{{Platform: platform, MatchType: "prefix", Model: model, Effort: effort}}}
}

func TestResolveReasoningEffortDefault(t *testing.T) {
	cfg := defaultReasoningTestConfig(PlatformOpenAI, "gpt-6.1-sol", "high")
	for _, protocol := range []string{"responses", "chat", "messages"} {
		path := map[string]string{"responses": "reasoning.effort", "chat": "reasoning_effort", "messages": "output_config.effort"}[protocol]
		for _, field := range []string{"", "null", `""`, `" "`, `"none"`, `"minimal"`, `"low"`, `"medium"`, `"high"`, `"xhigh"`, `"max"`, `"ultra"`, "true", "7", "[]", "{}"} {
			t.Run(protocol+"/"+field, func(t *testing.T) {
				req := map[string]any{"model": "gpt-6.1-sol"}
				if field != "" {
					var value any
					require.NoError(t, json.Unmarshal([]byte(field), &value))
					switch protocol {
					case "responses":
						req["reasoning"] = map[string]any{"effort": value, "summary": "auto"}
					case "chat":
						req["reasoning_effort"] = value
					case "messages":
						req["output_config"] = map[string]any{"effort": value, "format": map[string]any{"type": "text"}}
					}
				}
				body, err := json.Marshal(req)
				require.NoError(t, err)
				out, source, err := ResolveReasoningEffortDefault(body, protocol, PlatformOpenAI, "gpt-6.1-sol", cfg)
				if field == "true" || field == "7" || field == "[]" || field == "{}" {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				if field == "" || field == "null" || field == `""` || field == `" "` {
					require.Equal(t, ReasoningEffortSourceDefault, source)
					require.Equal(t, "high", gjson.GetBytes(out, path).String())
				} else {
					require.Equal(t, ReasoningEffortSourceExplicit, source)
					require.Equal(t, body, out, "never rewrite explicit client input")
				}
				if protocol == "messages" {
					require.False(t, gjson.GetBytes(out, "thinking").Exists())
				}
				if field != "" && protocol == "responses" {
					require.Equal(t, "auto", gjson.GetBytes(out, "reasoning.summary").String())
				}
			})
		}
	}
	for _, tt := range []struct {
		name, body, protocol, source string
		fail                         bool
	}{
		{"nested explicit", `{"reasoning":{"effort":"low"}}`, "chat", ReasoningEffortSourceExplicit, false},
		{"equal duplicate", `{"reasoning":{"effort":"HIGH"},"reasoning_effort":"high"}`, "chat", ReasoningEffortSourceExplicit, false},
		{"alias duplicate", `{"reasoning":{"effort":"extra_high"},"reasoning_effort":"xhigh"}`, "chat", ReasoningEffortSourceExplicit, false},
		{"conflict", `{"reasoning":{"effort":"low"},"reasoning_effort":"high"}`, "chat", "", true},
		{"thinking disabled", `{"thinking":{"type":"disabled"}}`, "messages", ReasoningEffortSourceExplicit, false},
		{"thinking conflict", `{"thinking":{"type":"disabled"},"output_config":{"effort":"high"}}`, "messages", "", true},
		{"thinking wrong type", `{"thinking":{"type":false}}`, "messages", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, source, err := ResolveReasoningEffortDefault([]byte(tt.body), tt.protocol, PlatformOpenAI, "gpt-6.1-sol", cfg)
			if tt.fail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.source, source)
			require.Equal(t, []byte(tt.body), out)
		})
	}
	for _, tt := range []struct{ name, model, platform, effort, source string }{
		{"suffix", "gpt-6.1-sol-high", PlatformOpenAI, "high", ReasoningEffortSourceModelSuffix},
		{"unknown model", "gpt-5", PlatformOpenAI, "high", ""},
		{"unknown matched model", "gpt-6.1-sol-unknown", PlatformOpenAI, "high", ""},
		{"unsupported minimal", "gpt-6.1-sol", PlatformOpenAI, "minimal", ""},
		{"unknown Claude", "claude-unknown", PlatformAnthropic, "high", ""},
		{"Claude no max", "claude-opus-4-5", PlatformAnthropic, "max", ""},
		{"Claude high", "claude-opus-4-5", PlatformAnthropic, "high", ReasoningEffortSourceDefault},
		{"platform mismatch", "gpt-6.1-sol", PlatformGemini, "high", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"model":"` + tt.model + `"}`)
			cfg := defaultReasoningTestConfig(PlatformComposite, "", tt.effort)
			out, source, err := ResolveReasoningEffortDefault(body, "responses", tt.platform, tt.model, cfg)
			require.NoError(t, err)
			require.Equal(t, tt.source, source)
			if source != ReasoningEffortSourceDefault {
				require.Equal(t, body, out)
			}
		})
	}
	t.Run("disabled config", func(t *testing.T) {
		cfg.Enabled = false
		body := []byte(`{"model":"gpt-6.1-sol"}`)
		out, source, err := ResolveReasoningEffortDefault(body, "responses", PlatformOpenAI, "gpt-6.1-sol", cfg)
		require.NoError(t, err)
		require.Empty(t, source)
		require.Equal(t, body, out)
	})
}

func TestReasoningEffortDefaultRuleSpecificity(t *testing.T) {
	cfg := ReasoningEffortDefaultConfig{Enabled: true, Rules: []ReasoningEffortDefaultRule{
		{Platform: PlatformOpenAI, Effort: "low"},
		{Platform: PlatformOpenAI, MatchType: "prefix", Model: "gpt", Effort: "medium"},
		{Platform: PlatformOpenAI, MatchType: "prefix", Model: "gpt-6.1", Effort: "high"},
		{Platform: PlatformOpenAI, MatchType: "exact", Model: "gpt-6.1-sol", Effort: "xhigh"},
	}}
	out, source, err := ResolveReasoningEffortDefault([]byte(`{"model":"gpt-6.1-sol"}`), "responses", PlatformOpenAI, "gpt-6.1-sol", cfg)
	require.NoError(t, err)
	require.Equal(t, ReasoningEffortSourceDefault, source)
	require.Equal(t, "xhigh", gjson.GetBytes(out, "reasoning.effort").String())
}

func TestReasoningEffortDefaultPolicyAndUsage(t *testing.T) {
	body := []byte(`{"model":"gpt-6.1-sol","reasoning":{"effort":"high","summary":"auto"}}`)
	out, changed, err := ApplyReasoningEffortPolicyWithSource(body, "medium", nil, ReasoningEffortOverLimitDeny, ReasoningEffortSourceDefault)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, "medium", gjson.GetBytes(out, "reasoning.effort").String())
	_, _, err = ApplyReasoningEffortPolicyWithSource(body, "medium", nil, ReasoningEffortOverLimitDeny, ReasoningEffortSourceExplicit)
	require.True(t, IsReasoningEffortPolicyDenied(err))
	mappings := []ReasoningEffortMapping{{From: "high", To: ReasoningEffortMappingDeny}}
	out, changed, err = ApplyReasoningEffortPolicyWithSource(body, "", mappings, "deny", ReasoningEffortSourceDefault)
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(out, "reasoning.effort").Exists())
	require.Equal(t, "auto", gjson.GetBytes(out, "reasoning.summary").String())
	requested, forwarded, source := "high", "medium", ReasoningEffortSourceDefault
	require.Nil(t, coalesceRequestedReasoningEffortWithSource(&requested, &forwarded, &source))
}

func TestNormalizeReasoningEffortDefaultConfig(t *testing.T) {
	for _, tt := range []struct {
		name, platform, match, model, effort string
		fail                                 bool
	}{
		{"OpenAI prefix", PlatformOpenAI, "prefix", "gpt-6.1-sol", "high", false},
		{"all known OpenAI", PlatformOpenAI, "", "", "high", false},
		{"Claude exact", PlatformAnthropic, "exact", "claude-opus-4-5", "high", false},
		{"Claude broad", PlatformAnthropic, "prefix", "claude-", "high", false},
		{"unsupported Claude max", PlatformAnthropic, "exact", "claude-opus-4-5", "max", true},
		{"unsupported Sol minimal", PlatformOpenAI, "prefix", "gpt-6.1-sol", "minimal", true},
		{"unknown exact", PlatformOpenAI, "exact", "unknown", "high", true},
		{"none", PlatformOpenAI, "prefix", "gpt", "none", true},
		{"platform", PlatformGemini, "prefix", "gemini", "high", true},
		{"match type", PlatformOpenAI, "regex", "gpt", "high", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeReasoningEffortDefaultConfig(ReasoningEffortDefaultConfig{Rules: []ReasoningEffortDefaultRule{{Platform: tt.platform, MatchType: tt.match, Model: tt.model, Effort: tt.effort}}})
			if tt.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
	cfg, err := ParseReasoningEffortDefaultConfig("")
	require.NoError(t, err)
	require.NotNil(t, cfg.Rules)
}
