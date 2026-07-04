package service

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

const usageSessionDisplayModulo = 999

type UsageSessionKey struct {
	Source string
	Value  string
}

func UsageSessionDisplayIndex(index int) int {
	if index <= 0 {
		return 0
	}
	return ((index - 1) % usageSessionDisplayModulo) + 1
}

func HashUsageSessionKey(source string, value string) string {
	source = strings.TrimSpace(source)
	value = strings.TrimSpace(value)
	if source == "" || value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(source + "\x00" + value))
	return hex.EncodeToString(sum[:])
}

func ExtractUsageSessionKey(body []byte, headers http.Header, responseID string) UsageSessionKey {
	for _, header := range []string{"X-Claude-Code-Session-Id", "session_id", "OpenAI-Conversation-ID"} {
		if value := strings.TrimSpace(headers.Get(header)); value != "" {
			return UsageSessionKey{Source: header, Value: value}
		}
	}
	if value := strings.TrimSpace(gjson.GetBytes(body, "metadata.user_id").String()); value != "" {
		return UsageSessionKey{Source: "metadata.user_id", Value: value}
	}
	if value := strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String()); value != "" {
		return UsageSessionKey{Source: "prompt_cache_key", Value: value}
	}
	if value := strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String()); value != "" {
		return UsageSessionKey{Source: "previous_response_id", Value: value}
	}
	if value := strings.TrimSpace(responseID); value != "" {
		return UsageSessionKey{Source: "response_id", Value: value}
	}
	return UsageSessionKey{}
}

func UsageSessionHashes(body []byte, headers http.Header, responseID string) (string, []string) {
	key := ExtractUsageSessionKey(body, headers, responseID)
	primary := HashUsageSessionKey(key.Source, key.Value)
	alias := HashUsageSessionKey("response_id", responseID)
	if alias == "" || alias == primary {
		return primary, nil
	}
	return primary, []string{alias}
}
