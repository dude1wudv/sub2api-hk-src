package service

import (
	"context"

	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
	"github.com/tidwall/gjson"
)

// Refuses a new WS response round once the account is paused mid-session.
// This decorator leaves the upstream relay package unchanged for upstream merges.
type slowTTFTFrameConn struct {
	openaiwsv2.FrameConn
	ctx        context.Context
	protection *RateLimitService
	accountID  int64
}

func (c *slowTTFTFrameConn) WriteFrame(ctx context.Context, t coderws.MessageType, p []byte) error {
	if c.protection != nil && gjson.GetBytes(p, "type").String() == "response.create" && c.protection.SlowTTFTPaused(c.ctx, c.accountID) {
		return ErrNoAvailableAccounts
	}
	return c.FrameConn.WriteFrame(ctx, t, p)
}
