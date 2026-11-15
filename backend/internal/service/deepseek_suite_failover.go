package service

import (
	"fmt"
	"net/http"

	"github.com/tidwall/gjson"
)

const DeepSeekSuiteFallbackAccountKey = "deepseek_suite_fallback_account_id"

// DeepSeekSuiteRejectedError is returned before writing any client response.
// Only explicit, unmetered rejections (or a WebSocket handshake failure before
// any text was sent) may be retried. Ambiguous transport errors stay pending.
type DeepSeekSuiteRejectedError struct {
	StatusCode int
	Body       []byte
}

func (e *DeepSeekSuiteRejectedError) Error() string {
	return fmt.Sprintf("DeepSeek upstream rejected operation (HTTP %d)", e.StatusCode)
}

func deepSeekSuiteRetryableRejection(status int, body []byte) bool {
	switch status {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusUnprocessableEntity,
		http.StatusTooManyRequests, http.StatusNotImplemented:
	default:
		return false
	}
	for _, field := range []string{"usage", "response.usage", "page_count", "duration",
		"num_chars", "id", "file_id", "server_tool_calls"} {
		if gjson.GetBytes(body, field).Exists() {
			return false
		}
	}
	return true
}

// The handler sends this ID through the ordinary scheduler and revalidates
// group membership, model support, capability and concurrency on the next turn.
// Existing file references must never move between provider accounts.
func (a *Account) DeepSeekSuiteFallbackAccountID(boundAccountID int64) int64 {
	if a == nil || boundAccountID != 0 {
		return 0
	}
	ids, valid := deepSeekToolsGroupIDs([]any{a.Extra[DeepSeekSuiteFallbackAccountKey]})
	if !valid || len(ids) != 1 || ids[0] == a.ID {
		return 0
	}
	return ids[0]
}
