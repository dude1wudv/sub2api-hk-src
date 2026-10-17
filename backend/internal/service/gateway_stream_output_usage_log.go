package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

// LogRecovery records only counters, never response content or upstream errors.
// It distinguishes a starting token with no received content from a stale usage
// snapshot followed by unmetered output, without retaining users' conversations.
func (u *anthropicStreamOutputUsage) LogRecovery(ctx context.Context, accountID int64, route string, usage *ClaudeUsage, terminal, disconnected bool, streamErr error) {
	if u == nil || usage == nil || (streamErr == nil && !disconnected && !usage.OutputTokensEstimated) {
		return
	}
	logger.FromContext(ctx).Info("gateway.stream_usage_recovery",
		zap.Int64("account_id", accountID),
		zap.String("stream_route", route),
		zap.Uint64("semantic_bytes", u.semanticBytes),
		zap.Int("initial_output_tokens", u.initialOutput),
		zap.Bool("initial_output_estimated", u.initialEstimated),
		zap.Bool("reported_usage", u.authoritative),
		zap.Int("reported_output_tokens", u.authoritativeOutput),
		zap.Uint64("reported_at_bytes", u.reportedAtBytes),
		zap.Bool("reported_estimated", u.reportedEstimated),
		zap.Int("output_tokens", usage.OutputTokens),
		zap.Bool("output_tokens_estimated", usage.OutputTokensEstimated),
		zap.Bool("terminal_seen", terminal),
		zap.Bool("client_disconnected", disconnected),
		zap.Bool("stream_error", streamErr != nil),
	)
}
