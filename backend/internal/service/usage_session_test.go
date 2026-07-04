package service

import (
	"net/http"
	"testing"
)

func TestUsageSessionDisplayIndexWrapsAfter999(t *testing.T) {
	cases := map[int]int{0: 0, 1: 1, 999: 999, 1000: 1, 1001: 2}
	for input, want := range cases {
		if got := UsageSessionDisplayIndex(input); got != want {
			t.Fatalf("UsageSessionDisplayIndex(%d) = %d, want %d", input, got, want)
		}
	}
}

func TestHashUsageSessionKeyIsStableAndScoped(t *testing.T) {
	a := HashUsageSessionKey("metadata", "abc")
	b := HashUsageSessionKey("metadata", "abc")
	c := HashUsageSessionKey("prompt_cache_key", "abc")
	if a == "" || a != b || a == c {
		t.Fatalf("unexpected hashes: a=%q b=%q c=%q", a, b, c)
	}
}

func TestExtractUsageSessionKeyPriority(t *testing.T) {
	headers := http.Header{}
	body := []byte(`{"metadata":{"user_id":"meta-1"},"prompt_cache_key":"cache-1","previous_response_id":"resp_prev"}`)
	key := ExtractUsageSessionKey(body, headers, "")
	if key.Source != "metadata.user_id" || key.Value != "meta-1" {
		t.Fatalf("key = %#v", key)
	}
}

func TestExtractUsageSessionKeyFallsBackToResponseID(t *testing.T) {
	key := ExtractUsageSessionKey([]byte(`{"model":"gpt-5.5"}`), http.Header{}, "resp_123")
	if key.Source != "response_id" || key.Value != "resp_123" {
		t.Fatalf("key = %#v", key)
	}
}
