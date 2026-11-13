//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type chatRemoteRepoFake struct {
	ChatRepository
	mu   sync.Mutex
	file ChatAttachmentRecord
	rows map[int64]ChatRemoteFileRecord
}

func (r *chatRemoteRepoFake) GetAttachment(_ context.Context, owner int64, id uuid.UUID) (*ChatAttachmentRecord, error) {
	if owner != r.file.UserID || id != r.file.ID {
		return nil, ErrChatNotFound
	}
	copy := r.file
	return &copy, nil
}
func (r *chatRemoteRepoFake) ClaimRemoteFile(ctx context.Context, owner int64, id uuid.UUID, account int64) (*ChatRemoteFileRecord, bool, error) {
	if _, err := r.GetAttachment(ctx, owner, id); err != nil {
		return nil, false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	row, exists := r.rows[account]
	if exists && row.Status == "error" {
		return nil, false, ErrChatRemoteFile
	}
	claimed := !exists || row.Status == "invalid"
	if claimed {
		row = ChatRemoteFileRecord{ID: int64(len(r.rows) + 1), UserID: owner, AttachmentID: id, AccountID: account, Status: "uploading", UpdatedAt: time.Now()}
		r.rows[account] = row
	}
	return &row, claimed, nil
}
func (r *chatRemoteRepoFake) FinishRemoteFile(_ context.Context, claim *ChatRemoteFileRecord, remote string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[claim.AccountID]
	if row.Status != "uploading" || !row.UpdatedAt.Equal(claim.UpdatedAt) {
		return ErrChatLeaseLost
	}
	row.RemoteID, row.Status = remote, "ready"
	if remote == "" {
		row.Status = "error"
	}
	r.rows[claim.AccountID] = row
	return nil
}
func (r *chatRemoteRepoFake) InvalidateRemoteFiles(_ context.Context, owner, account int64, ids []uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[account]
	for _, id := range ids {
		if row.UserID == owner && row.AttachmentID == id && row.Status == "ready" {
			row.Status = "invalid"
			r.rows[account] = row
		}
	}
	return nil
}
func chatNativeFileFixture(t *testing.T) (*OpenAIGatewayService, *chatNativeRequest, []byte) {
	t.Helper()
	id := uuid.New()
	content := "%PDF-1.7 test document"
	storage, err := NewChatStorage(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })
	_, err = storage.WriteFile(context.Background(), 7, id, strings.NewReader(content), 1000)
	require.NoError(t, err)
	repo := &chatRemoteRepoFake{file: ChatAttachmentRecord{ID: id, UserID: 7, Name: "test.pdf", Size: int64(len(content)), MIME: "application/pdf", Purpose: "rich_layout_recognition", Status: "ready"}, rows: map[int64]ChatRemoteFileRecord{}}
	native := &chatNativeRequest{service: &ChatService{repo: repo, storage: storage}, userID: 7, groupID: 1, files: map[uuid.UUID]bool{id: true}}
	body := []byte(fmt.Sprintf(`{"model":"deepseek-test","max_tokens":32,"stream":false,"messages":[{"role":"user","content":[{"type":"document","source":{"type":"file","file_id":"sub2api-chat:%s"}}]}]}`, id))
	return &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig()}, native, body
}
func chatNativeFileAccount(id int64) *Account {
	account := nativeAnthropicTestAccount()
	account.ID, account.Platform = id, PlatformDeepseek
	account.Credentials["api_key"] = fmt.Sprintf("test-account-%d", id)
	account.Credentials["api_base_urls"] = map[string]any{APIProtocolAnthropic: "http://anthropic.example/anthropic"}
	return account
}
func TestChatNativeFilesUseSelectedAccountAndPrivateContext(t *testing.T) {
	gateway, native, body := chatNativeFileFixture(t)
	ctx := context.WithValue(context.Background(), chatNativeContextKey{}, native)
	calls := map[int64]int{}
	gateway.httpUpstream = &codexModelsHTTPUpstreamStub{do: func(request *http.Request, proxy string, account int64, _ int) (*http.Response, error) {
		calls[account]++
		require.Equal(t, "/anthropic/v1/files", request.URL.Path)
		require.Empty(t, proxy)
		require.Equal(t, fmt.Sprintf("test-account-%d", account), getHeaderRaw(request.Header, "x-api-key"))
		require.Empty(t, request.Header.Get("Cookie"))
		mediaType, params, err := mime.ParseMediaType(getHeaderRaw(request.Header, "content-type"))
		require.NoError(t, err)
		require.Equal(t, "multipart/form-data", mediaType)
		reader := multipart.NewReader(request.Body, params["boundary"])
		purpose, err := reader.NextPart()
		require.NoError(t, err)
		value, err := io.ReadAll(purpose)
		require.NoError(t, err)
		require.Equal(t, "rich_layout_recognition", string(value))
		file, err := reader.NextPart()
		require.NoError(t, err)
		require.Equal(t, "test.pdf", file.FileName())
		value, err = io.ReadAll(file)
		require.NoError(t, err)
		require.Contains(t, string(value), "%PDF-")
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"file_%d"}`, account)))}, nil
	}}
	unchanged, err := gateway.PrepareChatAttachments(context.Background(), chatNativeFileAccount(1), body)
	require.NoError(t, err)
	require.Equal(t, body, unchanged)
	require.Empty(t, calls)
	for _, id := range []int64{1, 2, 1} {
		prepared, err := gateway.PrepareChatAttachments(ctx, chatNativeFileAccount(id), body)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("file_%d", id), gjson.GetBytes(prepared, "messages.0.content.0.source.file_id").String())
	}
	require.Equal(t, map[int64]int{1: 1, 2: 1}, calls)
	native.userID = 8
	_, err = gateway.PrepareChatAttachments(ctx, chatNativeFileAccount(3), body)
	require.ErrorIs(t, err, ErrChatNotFound)
	require.Zero(t, calls[3])
}
func TestChatNativeFilesRejectIncompatibleProtocolBeforeIO(t *testing.T) {
	gateway, native, body := chatNativeFileFixture(t)
	ctx := context.WithValue(context.Background(), chatNativeContextKey{}, native)
	account := chatNativeFileAccount(1)
	account.Credentials["api_protocol"] = APIProtocolChatCompletions
	gateway.httpUpstream = &codexModelsHTTPUpstreamStub{do: func(*http.Request, string, int64, int) (*http.Response, error) {
		t.Fatal("incompatible request must not leave gateway")
		return nil, nil
	}}
	_, err := gateway.ForwardAsAnthropic(ctx, adaptiveProtocolTestContext("/v1/messages", body), account, body, "", "")
	require.ErrorIs(t, err, ErrChatCapability)
}
func TestChatNativeFilesRetryOnlyExplicitExpiryOnce(t *testing.T) {
	for _, alwaysExpired := range []bool{false, true} {
		t.Run(fmt.Sprint(alwaysExpired), func(t *testing.T) {
			gateway, native, body := chatNativeFileFixture(t)
			ctx := context.WithValue(context.Background(), chatNativeContextKey{}, native)
			uploads, inferences := 0, 0
			gateway.httpUpstream = &codexModelsHTTPUpstreamStub{do: func(request *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
				if strings.HasSuffix(request.URL.Path, "/files") {
					uploads++
					_, _ = io.Copy(io.Discard, request.Body)
					return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"file_%d"}`, uploads)))}, nil
				}
				inferences++
				sent, err := io.ReadAll(request.Body)
				require.NoError(t, err)
				require.Equal(t, fmt.Sprintf("file_%d", uploads), gjson.GetBytes(sent, "messages.0.content.0.source.file_id").String())
				if inferences == 1 || alwaysExpired {
					return &http.Response{StatusCode: 400, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","code":"file_expired"}}`))}, nil
				}
				return nativeAnthropicBufferedResponse(), nil
			}}
			c := adaptiveProtocolTestContext("/v1/messages", body)
			c.Request.Header.Set("anthropic-beta", "files-api-2025-04-14")
			_, err := gateway.ForwardAsAnthropic(ctx, c, chatNativeFileAccount(1), body, "", "")
			if alwaysExpired {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 2, uploads)
			require.Equal(t, 2, inferences)
		})
	}
	require.False(t, chatExplicitlyExpiredFile([]byte(`{"error":{"type":"invalid_request_error","message":"unknown failure"}}`)))
}
func TestChatNativeFilesConcurrentClaimAndUnknownOutcome(t *testing.T) {
	_, native, _ := chatNativeFileFixture(t)
	var id uuid.UUID
	for value := range native.files {
		id = value
	}
	var calls atomic.Int32
	start, finish := make(chan struct{}), make(chan struct{})
	upload := func(context.Context) (string, error) {
		calls.Add(1)
		close(start)
		<-finish
		return "file_only_once", nil
	}
	errors := make(chan error, 8)
	var workers sync.WaitGroup
	for range 8 {
		workers.Go(func() { _, err := native.service.remoteFile(context.Background(), 7, id, 1, upload); errors <- err })
	}
	<-start
	close(finish)
	workers.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), calls.Load())
	failed := 0
	unknown := func(context.Context) (string, error) { failed++; return "", io.ErrUnexpectedEOF }
	_, err := native.service.remoteFile(context.Background(), 7, id, 2, unknown)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	_, err = native.service.remoteFile(context.Background(), 7, id, 2, unknown)
	require.ErrorIs(t, err, ErrChatRemoteFile)
	require.Equal(t, 1, failed)
}
