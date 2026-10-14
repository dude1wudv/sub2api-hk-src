package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// UsesGeminiChatCompletions opts a Gemini API-key account into forwarding Chat
// Completions requests directly to its OpenAI-compatible endpoint. Some relays
// expose both protocols but lose function calls in their Gemini translation.
// Native Gemini requests and accounts without this explicit flag keep their
// existing protocol. Require an explicit base URL; never fall back to OpenAI.
func (a *Account) UsesGeminiChatCompletions() bool {
	if a == nil || a.Platform != PlatformGemini || a.Type != AccountTypeAPIKey {
		return false
	}
	enabled, _ := a.Extra["gemini_chat_completions_passthrough"].(bool)
	return enabled && strings.TrimSpace(a.GetCredential("base_url")) != ""
}

// ForwardGeminiChatCompletions reuses the established raw Chat pipeline so
// tool-call deltas, finish reasons, tool results, error handling, and usage are
// preserved without a round trip through Gemini/Anthropic/Responses formats.
func (s *OpenAIGatewayService) ForwardGeminiChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
) (*ForwardResult, error) {
	if !account.UsesGeminiChatCompletions() {
		writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", "Gemini Chat Completions passthrough is not enabled for this account")
		return nil, fmt.Errorf("gemini chat completions passthrough is not enabled")
	}
	beginUpstreamResponseModelObservation(c)
	result, err := s.forwardAsRawChatCompletions(ctx, c, account, body, "")
	if result == nil {
		return nil, err
	}
	// Chat prompt_tokens includes cached tokens; the generic Gemini/Anthropic
	// billing path expects input tokens excluding separately billed cache reads
	// and writes, matching the existing OpenAI billing calculation.
	cached := max(0, min(result.Usage.CacheReadInputTokens, result.Usage.InputTokens))
	created := max(0, min(result.Usage.CacheCreationInputTokens, result.Usage.InputTokens-cached))
	return &ForwardResult{
		RequestID:                     result.RequestID,
		UpstreamHeaders:               result.UpstreamHeaders,
		Model:                         result.Model,
		UpstreamModel:                 result.UpstreamModel,
		UpstreamResponseModel:         result.UpstreamResponseModel,
		UpstreamResponseModelConflict: result.UpstreamResponseModelConflict,
		UpstreamResponseServiceTier:   result.UpstreamResponseServiceTier,
		Stream:                        result.Stream,
		Duration:                      result.Duration,
		FirstTokenMs:                  result.FirstTokenMs,
		ReasoningEffort:               result.ReasoningEffort,
		ServiceTier:                   result.ServiceTier,
		ClientDisconnect:              ctx.Err() != nil,
		Usage: ClaudeUsage{
			InputTokens:              max(0, result.Usage.InputTokens-cached-created),
			OutputTokens:             result.Usage.OutputTokens,
			CacheReadInputTokens:     cached,
			CacheCreationInputTokens: created,
			ImageOutputTokens:        result.Usage.ImageOutputTokens,
		},
	}, err
}
