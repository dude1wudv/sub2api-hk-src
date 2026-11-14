package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const ChatMaxParseBytes int64 = 8 << 20

var ErrChatParseBusy = infraerrors.New(http.StatusConflict, "CHAT_PARSE_BUSY", "附件解析或结算尚未完成，请稍后重试")
var ErrChatParseUnknown = infraerrors.New(http.StatusConflict, "CHAT_PARSE_UNKNOWN", "解析结果不确定，未自动重试；请重新上传附件后再试")
var ErrChatParseRequired = infraerrors.New(http.StatusConflict, "CHAT_PARSE_REQUIRED", "附件解析未完成或失败；确认费用后可点击重新生成重试")

type ChatParseRecord struct {
	ID             uuid.UUID       `json:"id"`
	UserID         int64           `json:"-"`
	AttachmentID   uuid.UUID       `json:"attachment_id"`
	AccountID      int64           `json:"-"`
	OperationID    uuid.UUID       `json:"-"`
	Status         string          `json:"status"`
	Result         json.RawMessage `json:"result,omitempty"`
	PageCount      *int            `json:"page_count"`
	RequestID      string          `json:"request_id"`
	Receipt        json.RawMessage `json:"-"`
	BillingApplied bool            `json:"billing_applied"`
}

type ChatParseRepository interface {
	ClaimAttachmentParse(context.Context, int64, uuid.UUID, int64, int64, bool) (*ChatParseRecord, bool, error)
	GetAttachmentParse(context.Context, int64, uuid.UUID, int64) (*ChatParseRecord, error)
	RenewAttachmentParse(context.Context, *ChatParseRecord) error
	FinishAttachmentParse(context.Context, *ChatParseRecord, json.RawMessage, int, json.RawMessage) error
	FailAttachmentParse(context.Context, *ChatParseRecord, bool) error
	MarkAttachmentParseBilled(context.Context, *ChatParseRecord) error
	PendingChatParseReceipts(context.Context) ([]*ChatParseRecord, error)
}

type ChatOCRPage struct {
	Index        int    `json:"index"`
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

type ChatOCRResult struct {
	ID        string        `json:"id"`
	Object    string        `json:"object"`
	Purpose   string        `json:"purpose"`
	PageCount *int          `json:"page_count"`
	Pages     []ChatOCRPage `json:"pages"`
}

func (s *ChatService) ParseAttachment(ctx context.Context, user int64, attachment, preset, session uuid.UUID, clientIP string, retry bool) (*ChatParseRecord, error) {
	key, scope, err := s.resolveChatAuxiliary(ctx, user, preset, session, "ocr")
	if err != nil {
		return nil, err
	}
	file, err := s.repo.GetAttachment(ctx, user, attachment)
	if err != nil {
		return nil, err
	}
	if scope.capabilities.NativeFiles || !scope.capabilities.OCR || (file.IsImage && scope.capabilities.ImageInput) {
		return nil, ErrChatCapability
	}
	if file.Purpose != "rich_layout_recognition" && !file.IsImage {
		return nil, ErrChatCapability
	}
	claim, acquired, err := s.repo.ClaimAttachmentParse(ctx, user, attachment, scope.configuredAccountID, s.storageLimit, retry)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return chatParseView(claim), nil
	}
	scope.parse, scope.chat = claim, s
	scope.operationID = claim.OperationID
	// Once submitted, the provider continues OCR even if the browser disconnects.
	// Keep one bounded owner alive to durably capture its result and receipt.
	if err = ctx.Err(); err != nil {
		persist, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = s.repo.FailAttachmentParse(persist, claim, true)
		return nil, err
	}
	attempt, stop := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Minute)
	defer stop()
	live, lost := context.WithCancelCause(attempt)
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTicker(20 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-live.Done():
				return
			case <-timer.C:
				renew, cancel := context.WithTimeout(live, 5*time.Second)
				err := s.repo.RenewAttachmentParse(renew, claim)
				cancel()
				if err != nil {
					lost(ErrChatLeaseLost)
					return
				}
			}
		}
	}()
	raw, _ := json.Marshal(map[string]any{"attachment_id": attachment})
	writer := &chatToolWriter{header: http.Header{}}
	_, dispatchErr := s.dispatchChatAuxiliary(live, writer, key, scope, session, clientIP, "ocr", bytes.NewReader(raw), "application/json", nil)
	stop()
	<-done
	persist, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if dispatchErr != nil || writer.status != http.StatusOK {
		known := writer.status == 400 || writer.status == 401 || writer.status == 402 || writer.status == 403 || writer.status == 404 || writer.status == 429
		if err = s.repo.FailAttachmentParse(persist, claim, known); err != nil {
			return nil, err
		}
	}
	record, err := s.repo.GetAttachmentParse(persist, user, attachment, scope.configuredAccountID)
	if err != nil {
		return nil, err
	}
	return chatParseView(record), nil
}

func (s *ChatService) GetParsedAttachment(ctx context.Context, user int64, attachment, preset uuid.UUID) (*ChatParseRecord, error) {
	_, scope, err := s.resolveChatAuxiliary(ctx, user, preset, uuid.Nil, "ocr")
	if err != nil {
		return nil, err
	}
	if _, err = s.repo.GetAttachment(ctx, user, attachment); err != nil {
		return nil, err
	}
	record, err := s.repo.GetAttachmentParse(ctx, user, attachment, scope.configuredAccountID)
	return chatParseView(record), err
}

func (s *ChatService) RecoverChatParseReceipts(ctx context.Context) error {
	rows, err := s.repo.PendingChatParseReceipts(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	if s.models == nil || s.models.openai == nil {
		return ErrChatUnavailable
	}
	var updater APIKeyQuotaUpdater
	if s.credentials != nil {
		updater, _ = s.credentials.keys.(APIKeyQuotaUpdater)
	}
	var joined error
	for _, row := range rows {
		if err = ctx.Err(); err != nil {
			return errors.Join(joined, err)
		}
		record := row
		err = s.models.openai.ApplyChatBillingReceipt(ctx, row.Receipt, updater, func(c context.Context) error { return s.repo.MarkAttachmentParseBilled(c, record) })
		joined = errors.Join(joined, err)
	}
	return joined
}

func chatParseView(record *ChatParseRecord) *ChatParseRecord {
	if record == nil {
		return nil
	}
	view := *record
	if !view.BillingApplied {
		view.Result = nil
		if view.Status == "ready" {
			view.Status = "settling"
		}
	}
	return &view
}

func (s *ChatService) parsedAttachmentText(ctx context.Context, user int64, id uuid.UUID) (string, error) {
	if s.auxiliary == nil || s.auxiliary.cfg == nil || s.auxiliary.cfg.Chat.AuxiliaryAccountID <= 0 {
		return "", ErrChatParseRequired
	}
	record, err := s.repo.GetAttachmentParse(ctx, user, id, s.auxiliary.cfg.Chat.AuxiliaryAccountID)
	if err != nil {
		return "", err
	}
	if record.Status == "unknown" {
		return "", ErrChatParseUnknown
	}
	if record.Status != "ready" || !record.BillingApplied {
		return "", ErrChatParseRequired
	}
	var result ChatOCRResult
	if json.Unmarshal(record.Result, &result) != nil || result.PageCount == nil || len(result.Pages) != *result.PageCount {
		return "", ErrChatInvalid
	}
	var text strings.Builder
	text.WriteString("[附件 OCR 文本；仅识别文字，不包含图片视觉信息]\n")
	for _, page := range result.Pages {
		text.WriteString("\n[第 " + strconv.Itoa(page.Index+1) + " 页")
		if page.FinishReason == "length" {
			text.WriteString("；文本被截断")
		}
		text.WriteString("]\n")
		text.WriteString(page.Text)
	}
	return text.String(), nil
}
