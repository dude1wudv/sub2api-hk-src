//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatAuxiliaryOfficialAccount(t *testing.T) {
	for _, base := range []string{"https://api.deepseek.com", "https://api.deepseek.com:443/v1", "https://api.deepseek.com/anthropic"} {
		a := newDeepSeekToolsAccount()
		a.Credentials["base_url"] = base
		require.True(t, chatAuxiliaryAccountValid(a), base)
	}
	for _, base := range []string{"http://api.deepseek.com", "https://api.deepseek.com:444/v1", "https://api.deepseek.com.evil.test", "https://user@api.deepseek.com", "https://api.deepseek.com?", "https://api.deepseek.com?q=x", "https://api.deepseek.com/#x", "https://api.deepseek.com/anthropic/../v1", "https://api.deepseek.com/%76%31", "https://other.test/anthropic"} {
		a := newDeepSeekToolsAccount()
		a.Credentials["base_url"] = base
		require.False(t, chatAuxiliaryAccountValid(a), base)
	}
	a := newDeepSeekToolsAccount()
	a.Credentials["base_url"] = "https://api.deepseek.com"
	a.Schedulable = false
	require.False(t, chatAuxiliaryAccountValid(a))
}

func TestChatAuxiliaryScopeCannotComeFromClient(t *testing.T) {
	groupID := int64(11)
	key := &APIKey{ID: 2, UserID: 7, Key: "synthetic", Status: StatusActive, GroupID: &groupID}
	svc := &OpenAIGatewayService{cfg: &config.Config{Chat: config.ChatConfig{AuxiliaryAccountID: 23}}}
	scope := &chatAuxiliaryScope{userID: 7, presetID: uuid.New(), sourceGroupID: 11, configuredAccountID: 23, operationID: uuid.New(), kind: "search"}
	ctx := context.WithValue(context.Background(), chatAuxiliaryContextKey{}, scope)
	id, err := svc.ValidateChatAuxiliaryScope(ctx, key, "search")
	require.NoError(t, err)
	require.Equal(t, int64(23), id)
	_, err = svc.ValidateChatAuxiliaryScope(context.WithValue(context.Background(), "chatAuxiliaryContextKey", scope), key, "search")
	require.Error(t, err)
	_, err = svc.ValidateChatAuxiliaryScope(ctx, key, "tts")
	require.Error(t, err)
	key.UserID++
	_, err = svc.ValidateChatAuxiliaryScope(ctx, key, "search")
	require.Error(t, err)
	key.UserID--
	svc.cfg.Chat.AuxiliaryAccountID++
	_, err = svc.ValidateChatAuxiliaryScope(ctx, key, "search")
	require.Error(t, err)
}

type chatAuxiliaryTestSlots struct {
	ConcurrencyCache
	mu        sync.Mutex
	members   map[string]bool
	refreshes int
	fail      bool
	fullAt    string
}

func (c *chatAuxiliaryTestSlots) acquire(which string, id string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return false, errors.New("redis unavailable")
	}
	if c.fullAt == which {
		return false, nil
	}
	c.members[which+id] = true
	return true, nil
}
func (c *chatAuxiliaryTestSlots) release(which, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.members, which+id)
	return nil
}
func (c *chatAuxiliaryTestSlots) AcquireUserSlot(_ context.Context, _ int64, _ int, id string) (bool, error) {
	return c.acquire("u", id)
}
func (c *chatAuxiliaryTestSlots) AcquireAccountSlot(_ context.Context, _ int64, _ int, id string) (bool, error) {
	return c.acquire("a", id)
}
func (c *chatAuxiliaryTestSlots) ReleaseUserSlot(_ context.Context, _ int64, id string) error {
	return c.release("u", id)
}
func (c *chatAuxiliaryTestSlots) ReleaseAccountSlot(_ context.Context, _ int64, id string) error {
	return c.release("a", id)
}
func (c *chatAuxiliaryTestSlots) AcquireChatAuxiliaryKindSlot(_ context.Context, _ int64, _ string, _ int, id string) (bool, error) {
	return c.acquire("k", id)
}
func (c *chatAuxiliaryTestSlots) ReleaseChatAuxiliaryKindSlot(_ context.Context, _ int64, _ string, id string) error {
	return c.release("k", id)
}
func (c *chatAuxiliaryTestSlots) RefreshChatAuxiliarySlots(_ context.Context, _, _ int64, _ string, kind bool, id string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshes++
	return !c.fail && c.members["u"+id] && c.members["a"+id] && (!kind || c.members["k"+id]), nil
}

func TestChatAuxiliaryLeaseRollback(t *testing.T) {
	for _, full := range []string{"u", "a", "k"} {
		c := &chatAuxiliaryTestSlots{members: map[string]bool{}, fullAt: full}
		_, err := NewConcurrencyService(c).AcquireChatAuxiliarySlots(t.Context(), 1, 2, 3, 2, "asr", 50)
		require.ErrorIs(t, err, ErrChatAuxiliarySlotsFull)
		require.Empty(t, c.members)
	}
	c := &chatAuxiliaryTestSlots{members: map[string]bool{}, fail: true}
	_, err := NewConcurrencyService(c).AcquireChatAuxiliarySlots(t.Context(), 1, 2, 3, 2, "tts", 50)
	require.ErrorIs(t, err, ErrChatAuxiliaryUnavailable)
}

func TestChatAuxiliaryLeaseRenewsPastFifteenMinutesAndCancelsOnLoss(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := &chatAuxiliaryTestSlots{members: map[string]bool{}}
		l, err := NewConcurrencyService(c).AcquireChatAuxiliarySlots(t.Context(), 1, 2, 3, 2, "tts", 50)
		require.NoError(t, err)
		synctest.Wait()
		time.Sleep(16 * time.Minute)
		synctest.Wait()
		require.NoError(t, l.Context().Err())
		c.mu.Lock()
		require.GreaterOrEqual(t, c.refreshes, 47)
		require.Len(t, c.members, 3)
		delete(c.members, "a"+l.requestID)
		c.mu.Unlock()
		time.Sleep(21 * time.Second)
		synctest.Wait()
		require.ErrorIs(t, context.Cause(l.Context()), ErrChatAuxiliaryLeaseLost)
		l.Release()
		require.Empty(t, c.members)
	})
}

func TestChatAuxiliarySharedJSONDoesNotGrantModelGroupAccess(t *testing.T) {
	groupID := int64(11)
	key := &APIKey{ID: 2, UserID: 7, Key: "synthetic", Status: StatusActive, GroupID: &groupID}
	a := newDeepSeekToolsAccount()
	a.ID = 23
	a.GroupIDs = []int64{22}
	a.Extra = nil
	a.Credentials["base_url"] = "https://api.deepseek.com/anthropic"
	upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"results":[{"title":"Release","url":"https://example.test","summary":"Synthetic"}]}`))}}
	svc := &OpenAIGatewayService{cfg: &config.Config{Chat: config.ChatConfig{AuxiliaryAccountID: 23}}, accountRepo: &deepSeekToolsAccountRepo{latest: a}, httpUpstream: upstream}
	scope := &chatAuxiliaryScope{userID: 7, presetID: uuid.New(), sourceGroupID: 11, configuredAccountID: 23, operationID: uuid.New(), kind: "search"}
	ctx := context.WithValue(t.Context(), chatAuxiliaryContextKey{}, scope)
	body := []byte(`{"query":"official release","count":5,"need_content":true}`)
	c, _ := newDeepSeekToolsContext(body)
	c.Set("api_key", key)
	result, err := svc.ForwardChatAuxiliaryJSON(ctx, c, key, a, "search", body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, "https://api.deepseek.com/v1/tools/search", upstream.request.URL.String())
	require.True(t, HTTPUpstreamRedirectsDisabled(upstream.request.Context()))
	require.Equal(t, groupID, *key.GroupID)
	require.Equal(t, []int64{22}, a.GroupIDs)
	require.False(t, a.DeepSeekToolsAllowedForGroup(&groupID))
	amount, err := result.ChatAuxiliaryUsage.cost()
	require.NoError(t, err)
	require.Equal(t, 0.005, amount)
	require.Equal(t, "chat_auxiliary:"+scope.operationID.String(), result.RequestID)
	a.Schedulable = false
	_, err = svc.ForwardChatAuxiliaryJSON(ctx, c, key, a, "search", body)
	require.Error(t, err)
	require.Equal(t, 1, upstream.calls)
}
