package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMirasimNormalizeClaudeMessages(t *testing.T) {
	body := []byte(`{"model":"mirasim/claude-sonnet-5","top_p":0.9,"metadata":{"user_id":"u1"},"system":"follow the user","messages":[{"role":"system","content":"be concise"},{"role":"user","content":"hi"}]}`)
	encoded, err := normalizeMirasimBody("/v1/messages", body)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "claude-sonnet-5" || got["top_p"] != nil || got["metadata"] == nil {
		t.Fatalf("model/unsupported field/metadata normalization failed: %s", encoded)
	}
	msgs, ok := got["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("system message was not hoisted: %s", encoded)
	}
	first, ok := msgs[0].(map[string]any)
	if !ok || first["role"] != "user" {
		t.Fatalf("system message was not hoisted: %s", encoded)
	}
	system, ok := got["system"].([]any)
	if !ok {
		t.Fatalf("system blocks missing: %s", encoded)
	}
	total := 0
	for _, block := range system {
		value, ok := block.(map[string]any)
		if !ok {
			t.Fatalf("system block is not an object: %s", encoded)
		}
		text, ok := value["text"].(string)
		if !ok {
			t.Fatalf("system block text missing: %s", encoded)
		}
		total += len(text)
	}
	if total > 200 || !mirasimHasFingerprint(system) || !strings.Contains(string(encoded), "follow the user") || !strings.Contains(string(encoded), "be concise") {
		t.Fatalf("Claude fingerprint, instructions, or byte limit failed: %s", encoded)
	}
}

func TestMirasimNormalizeClaudeMessagesRejectsLongSystemWithoutTruncating(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-5","system":"` + strings.Repeat("中", 100) + `","messages":[{"role":"user","content":"hi"}]}`)
	if encoded, err := normalizeMirasimBody("/v1/messages", body); err == nil || encoded != nil || !strings.Contains(err.Error(), "not truncated") {
		t.Fatalf("expected explicit long-system error with no rewritten body, got body=%s err=%v", encoded, err)
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
	input, ok := got["input"].([]any)
	if got["model"] != "gpt-6-astra" || !ok || len(input) != 1 {
		t.Fatalf("Responses input was not wrapped: %s", encoded)
	}
	if _, err := normalizeMirasimBody("/v1/responses", []byte(`{"model":"gpt-6-astra","stream":false,"input":"hello"}`)); err == nil {
		t.Fatal("non-streaming GPT request must be rejected")
	}
	if _, err := normalizeMirasimBody("/v1/messages", []byte(`{"model":"gpt-6-astra","messages":[]}`)); err == nil {
		t.Fatal("GPT must not be routed to Messages")
	}
}
