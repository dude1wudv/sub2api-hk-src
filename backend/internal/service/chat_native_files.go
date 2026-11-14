package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

var ErrChatRemoteFile = infraerrors.BadRequest("CHAT_REMOTE_FILE_UNAVAILABLE", "附件上传失败或结果不确定，请重新上传附件后再试")

type ChatRemoteFileRecord struct {
	ID              int64
	UserID          int64
	AttachmentID    uuid.UUID
	AccountID       int64
	RemoteID        string
	Status          string
	UpdatedAt       time.Time
	CleanupAttempts int
	LeaseToken      uuid.UUID
}

type ChatRemoteFileRepository interface {
	ClaimRemoteFile(context.Context, int64, uuid.UUID, int64) (*ChatRemoteFileRecord, bool, error)
	FinishRemoteFile(context.Context, *ChatRemoteFileRecord, string) error
	RenewRemoteFile(context.Context, *ChatRemoteFileRecord) error
	InvalidateRemoteFiles(context.Context, int64, int64, []uuid.UUID) error
}

func chatValidRemoteID(id string) bool {
	if len(id) == 0 || len(id) > 512 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func (s *ChatService) remoteFile(ctx context.Context, userID int64, id uuid.UUID, accountID int64, upload func(context.Context) (string, error)) (string, error) {
	for {
		claim, acquired, err := s.repo.ClaimRemoteFile(ctx, userID, id, accountID)
		if err != nil {
			return "", err
		}
		if claim == nil {
			return "", ErrChatRemoteFile
		}
		if !acquired && claim.Status == "ready" {
			if !chatValidRemoteID(claim.RemoteID) {
				return "", ErrChatRemoteFile
			}
			return claim.RemoteID, nil
		}
		if acquired {
			attempt, cancel := context.WithTimeout(ctx, 10*time.Minute)
			leaseCtx, lost := context.WithCancelCause(attempt)
			renewed := make(chan struct{})
			go func() {
				defer close(renewed)
				ticker := time.NewTicker(20 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-leaseCtx.Done():
						return
					case <-ticker.C:
						live, done := context.WithTimeout(leaseCtx, 5*time.Second)
						err := s.repo.RenewRemoteFile(live, claim)
						done()
						if err != nil {
							lost(ErrChatLeaseLost)
							return
						}
					}
				}
			}()
			remoteID, uploadErr := upload(leaseCtx)
			if context.Cause(leaseCtx) == ErrChatLeaseLost {
				uploadErr = ErrChatLeaseLost
			}
			cancel()
			<-renewed
			if uploadErr == nil && !chatValidRemoteID(remoteID) {
				uploadErr = ErrChatRemoteFile
			}
			if uploadErr != nil {
				remoteID = ""
			}
			persist, done := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			saveErr := s.repo.FinishRemoteFile(persist, claim, remoteID)
			done()
			if err = errors.Join(uploadErr, saveErr); err != nil {
				return "", err
			}
			return remoteID, nil
		}
		if claim.Status != "uploading" {
			return "", ErrChatRemoteFile
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
}

// Only private in-process context can activate this adapter. Public API bodies
// are returned byte-for-byte, including any text resembling an internal marker.
func (s *OpenAIGatewayService) PrepareChatAttachments(ctx context.Context, account *Account, body []byte) ([]byte, error) {
	native, _ := ctx.Value(chatNativeContextKey{}).(*chatNativeRequest)
	if native == nil || len(native.files) == 0 {
		return body, nil
	}
	if native.service == nil || native.userID <= 0 || account == nil || account.Type != AccountTypeAPIKey {
		return nil, ErrChatCapability
	}
	model := upstreamRoutingModel(account, body, "")
	if s.resolveUpstreamProtocolFor(ctx, account, APIProtocolAnthropic, model) != APIProtocolAnthropic || !chatNativeDeepSeekAccount(account) {
		return nil, ErrChatCapability
	}
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return nil, ErrChatInvalid
	}
	type reference struct {
		source map[string]any
		id     uuid.UUID
	}
	refs := []reference{}
	var visit func(any) error
	visit = func(value any) error {
		switch item := value.(type) {
		case map[string]any:
			if source, ok := item["source"].(map[string]any); ok && source["type"] == "file" {
				value := chatString(source["file_id"])
				if !strings.HasPrefix(value, "sub2api-chat:") {
					return ErrChatInvalid
				}
				id, err := uuid.Parse(strings.TrimPrefix(value, "sub2api-chat:"))
				if err != nil || !native.files[id] {
					return ErrChatNotFound
				}
				refs = append(refs, reference{source, id})
			}
			for _, child := range item {
				if err := visit(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range item {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(payload); err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, ErrChatInvalid
	}
	for _, ref := range refs {
		remoteID, err := native.service.remoteFile(ctx, native.userID, ref.id, account.ID, func(attempt context.Context) (string, error) {
			return s.uploadChatNativeFile(attempt, native, account, ref.id)
		})
		if err != nil {
			return nil, err
		}
		ref.source["file_id"] = remoteID
	}
	return json.Marshal(payload)
}

func (s *OpenAIGatewayService) chatNativeFilesURL(account *Account) (string, error) {
	target, err := s.nativeAnthropicTargetURL(account)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(target)
	if err != nil || !strings.HasSuffix(parsed.Path, "/messages") {
		return "", ErrChatCapability
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/messages") + "/files"
	parsed.RawPath, parsed.RawQuery, parsed.Fragment = "", "", ""
	return parsed.String(), nil
}

func (s *OpenAIGatewayService) chatNativeFileRequest(ctx context.Context, account *Account, target string) (*http.Request, error) {
	key := strings.TrimSpace(account.GetOpenAIProtocolAPIKey())
	if key == "" {
		return nil, ErrChatRemoteFile
	}
	request, _, err := s.buildNativeAnthropicUpstreamRequest(ctx, nil, account, []byte(`{}`), key, target)
	if err != nil {
		return nil, err
	}
	beta := getHeaderRaw(request.Header, "anthropic-beta")
	if !strings.Contains(beta, "files-api-2025-04-14") {
		if beta != "" {
			beta += ","
		}
		setHeaderRaw(request.Header, "anthropic-beta", beta+"files-api-2025-04-14")
	}
	return request, nil
}

func chatAccountProxy(account *Account) string {
	if account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func (s *OpenAIGatewayService) uploadChatNativeFile(ctx context.Context, native *chatNativeRequest, account *Account, id uuid.UUID) (string, error) {
	record, file, err := native.service.OpenAttachment(ctx, native.userID, id)
	if err != nil {
		return "", err
	}
	defer file.Close()
	target, err := s.chatNativeFilesURL(account)
	if err != nil {
		return "", err
	}
	request, err := s.chatNativeFileRequest(ctx, account, target)
	if err != nil {
		return "", err
	}
	purpose := record.Purpose
	if purpose == "image" {
		purpose = ""
	}
	body, contentType, length, err := chatAttachmentMultipart(record, file, purpose)
	if err != nil {
		return "", err
	}
	request.Body = io.NopCloser(body)
	request.GetBody = nil
	request.ContentLength = length
	setHeaderRaw(request.Header, "content-type", contentType)
	response, err := s.doOpenAIUpstream(request, chatAccountProxy(account), account)
	if err != nil {
		return "", ErrChatRemoteFile
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", ErrChatRemoteFile
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 128*1024+1))
	if err != nil || len(raw) > 128*1024 || !json.Valid(raw) {
		return "", ErrChatRemoteFile
	}
	remoteID := gjson.GetBytes(raw, "id").String()
	if !chatValidRemoteID(remoteID) {
		return "", ErrChatRemoteFile
	}
	return remoteID, nil
}

func chatExplicitlyExpiredFile(body []byte) bool {
	for _, path := range []string{"error.code", "error.type"} {
		switch gjson.GetBytes(body, path).String() {
		case "file_not_found", "file_expired", "invalid_file_id":
			return true
		}
	}
	return false
}

func invalidateChatRemoteFiles(ctx context.Context, account *Account) error {
	native, _ := ctx.Value(chatNativeContextKey{}).(*chatNativeRequest)
	if native == nil || native.service == nil {
		return ErrChatCapability
	}
	ids := make([]uuid.UUID, 0, len(native.files))
	for id := range native.files {
		ids = append(ids, id)
	}
	return native.service.repo.InvalidateRemoteFiles(ctx, native.userID, account.ID, ids)
}

func (s *OpenAIGatewayService) deleteChatNativeFile(ctx context.Context, accountID int64, remoteID string) error {
	if !chatValidRemoteID(remoteID) {
		return ErrChatRemoteFile
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return err
	}
	target, err := s.chatNativeFilesURL(account)
	if err != nil {
		return err
	}
	request, err := s.chatNativeFileRequest(ctx, account, target+"/"+remoteID)
	if err != nil {
		return err
	}
	request.Method, request.Body, request.GetBody, request.ContentLength = http.MethodDelete, nil, nil, 0
	response, err := s.doOpenAIUpstream(request, chatAccountProxy(account), account)
	if err != nil {
		return ErrChatRemoteFile
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	return ErrChatRemoteFile
}
