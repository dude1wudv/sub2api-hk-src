package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMirasimNormalizeClaudeMessages(t *testing.T) {
	body := []byte(`{"model":"mirasim/claude-sonnet-5","top_p":0.9,"system":"` + strings.Repeat("中", 100) + `","messages":[{"role":"system","content":"extra"},{"role":"user","content":"hi"}]}`)
	encoded, err := normalizeMirasimBody("/v1/messages", body)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "claude-sonnet-5" || got["top_p"] != nil {
		t.Fatalf("model/top_p normalization failed: %s", encoded)
	}
	msgs := got["messages"].([]any)
	if len(msgs) != 1 || msgs[0].(map[string]any)["role"] != "user" {
		t.Fatalf("system message was not hoisted: %s", encoded)
	}
	system := got["system"].([]any)
	total := 0
	for _, block := range system {
		total += len(block.(map[string]any)["text"].(string))
	}
	if total > 200 || !mirasimHasFingerprint(system) {
		t.Fatalf("Claude fingerprint or byte limit failed: %s", encoded)
	}
}

func TestMirasimNormalizeResponses(t *testing.T) {
	encoded, err := normalizeMirasimBody("/v1/responses", []byte(`{"model":"mirasim/gpt-6-astra","stream":true,"input":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "gpt-6-astra" || len(got["input"].([]any)) != 1 {
		t.Fatalf("Responses input was not wrapped: %s", encoded)
	}
	if _, err := normalizeMirasimBody("/v1/responses", []byte(`{"model":"gpt-6-astra","stream":false,"input":"hello"}`)); err == nil {
		t.Fatal("non-streaming GPT request must be rejected")
	}
	if _, err := normalizeMirasimBody("/v1/messages", []byte(`{"model":"gpt-6-astra","messages":[]}`)); err == nil {
		t.Fatal("GPT must not be routed to Messages")
	}
}
