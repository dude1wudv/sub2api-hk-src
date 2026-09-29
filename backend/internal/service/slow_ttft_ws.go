package service

import (
	"context"
	"sync"

	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
	"github.com/tidwall/gjson"
)

// This decorator leaves the upstream relay package unchanged for upstream merges.
type slowTTFTFrameConn struct {
	openaiwsv2.FrameConn
	ctx        context.Context
	protection *RateLimitService
	accountID  int64
	mu         sync.Mutex
	observer   *SlowTTFTObserver
}

func (c *slowTTFTFrameConn) finishObserver() {
	c.mu.Lock()
	o := c.observer
	c.observer = nil
	c.mu.Unlock()
	o.Close()
}
func (c *slowTTFTFrameConn) WriteFrame(ctx context.Context, t coderws.MessageType, p []byte) error {
	if gjson.GetBytes(p, "type").String() == "response.create" && c.protection != nil {
		if c.protection.SlowTTFTPaused(c.ctx, c.accountID) {
			return ErrNoAvailableAccounts
		}
		c.finishObserver()
		if !gjson.GetBytes(p, "tools.#(type==\"image_generation\")").Exists() {
			a, err := c.protection.accountRepo.GetByID(c.ctx, c.accountID)
			if err != nil {
				return err
			}
			o := c.protection.BeginSlowTTFT(c.ctx, a)
			c.mu.Lock()
			c.observer = o
			c.mu.Unlock()
		}
	}
	err := c.FrameConn.WriteFrame(ctx, t, p)
	if err != nil {
		c.finishObserver()
	}
	return err
}
func (c *slowTTFTFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	t, p, err := c.FrameConn.ReadFrame(ctx)
	c.mu.Lock()
	o := c.observer
	c.mu.Unlock()
	if SlowTTFTMeaningfulOutput(p) {
		o.FirstOutput()
	}
	typ := gjson.GetBytes(p, "type").String()
	if err != nil || typ == "response.completed" || typ == "response.failed" || typ == "response.incomplete" || typ == "error" {
		c.finishObserver()
	}
	return t, p, err
}
func (c *slowTTFTFrameConn) Close() error { c.finishObserver(); return c.FrameConn.Close() }
