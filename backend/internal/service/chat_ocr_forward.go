package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

func chatAttachmentMultipart(record *ChatAttachmentRecord, file io.Reader, purpose string) (io.Reader, string, int64, error) {
	var envelope bytes.Buffer
	writer := multipart.NewWriter(&envelope)
	if purpose != "" {
		if err := writer.WriteField("purpose", purpose); err != nil {
			return nil, "", 0, err
		}
	}
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": record.Name}))
	headers.Set("Content-Type", record.MIME)
	if _, err := writer.CreatePart(headers); err != nil {
		return nil, "", 0, err
	}
	prefix := bytes.Clone(envelope.Bytes())
	envelope.Reset()
	if err := writer.Close(); err != nil {
		return nil, "", 0, err
	}
	suffix := bytes.Clone(envelope.Bytes())
	return io.MultiReader(bytes.NewReader(prefix), file, bytes.NewReader(suffix)), writer.FormDataContentType(), int64(len(prefix)+len(suffix)) + record.Size, nil
}

func (s *OpenAIGatewayService) ForwardChatOCR(ctx context.Context, key *APIKey, account *Account) (result *OpenAIForwardResult, parsed *ChatOCRResult, err error) {
	account, err = s.chatSpeechAccount(ctx, key, account, "ocr")
	if err != nil {
		return nil, nil, err
	}
	scope, _ := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	if scope == nil || scope.chat == nil || scope.parse == nil || scope.parse.AccountID != account.ID || scope.parse.UserID != key.UserID {
		return nil, nil, ErrChatInvalid
	}
	knownFailure := false
	defer func() {
		if err != nil {
			persist, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = scope.chat.repo.FailAttachmentParse(persist, scope.parse, knownFailure)
		}
	}()
	record, file, err := scope.chat.OpenAttachment(ctx, scope.userID, scope.parse.AttachmentID)
	if err != nil {
		knownFailure = true
		return nil, nil, err
	}
	defer file.Close()
	body, contentType, length, err := chatAttachmentMultipart(record, file, "text_recognition")
	if err != nil {
		knownFailure = true
		return nil, nil, err
	}
	attempt, stop := context.WithTimeout(ctx, 185*time.Second)
	defer stop()
	request, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(attempt), http.MethodPost, "https://api.deepseek.com/v1/tools/ocr", body)
	if err != nil {
		knownFailure = true
		return nil, nil, err
	}
	request.ContentLength = length
	request.GetBody = nil
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+account.GetCredential("api_key"))
	started := time.Now()
	response, err := s.httpUpstream.Do(request, resolveAccountProxyURL(account), account.ID, account.Concurrency)
	if err != nil {
		return nil, nil, err
	}
	if response == nil || response.Body == nil {
		return nil, nil, ErrChatStream
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		knownFailure = true
		return nil, nil, ErrChatStream
	}
	raw, err := readChatBounded(response.Body, ChatMaxParseBytes)
	if err != nil {
		return nil, nil, err
	}
	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return nil, nil, ErrChatStream
	}
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		knownFailure = true
		return nil, nil, ErrChatStream
	}
	parsed = &ChatOCRResult{}
	if json.Unmarshal(raw, parsed) != nil || parsed.ID == "" || len(parsed.ID) > 128 || parsed.Object != "ocr.result" || parsed.Purpose != "text_recognition" || parsed.PageCount == nil || *parsed.PageCount < 0 || *parsed.PageCount > 1000 || len(parsed.Pages) != *parsed.PageCount {
		return nil, nil, ErrChatStream
	}
	for i, page := range parsed.Pages {
		if page.Index != i || (page.FinishReason != "stop" && page.FinishReason != "length") {
			return nil, nil, ErrChatStream
		}
	}
	quantity := int64(*parsed.PageCount)
	result = chatSpeechResult(ctx, "ocr", started, &quantity, "confirmed")
	return result, parsed, nil
}

type ChatOCRSettlement struct {
	gateway *OpenAIGatewayService
	repo    ChatParseRepository
	record  *ChatParseRecord
	receipt json.RawMessage
}

func (s *OpenAIGatewayService) CommitChatOCR(ctx context.Context, input *OpenAIRecordUsageInput, parsed *ChatOCRResult) (*ChatOCRSettlement, error) {
	scope, _ := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	if scope == nil || scope.chat == nil || scope.parse == nil || parsed == nil || parsed.PageCount == nil {
		return nil, ErrChatInvalid
	}
	receipt, err := s.PrepareChatAuxiliaryBilling(ctx, input)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(parsed)
	if err != nil {
		return nil, err
	}
	persist, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err = scope.chat.repo.FinishAttachmentParse(persist, scope.parse, body, *parsed.PageCount, receipt); err != nil {
		return nil, err
	}
	return &ChatOCRSettlement{gateway: s, repo: scope.chat.repo, record: scope.parse, receipt: receipt}, nil
}

func (s *ChatOCRSettlement) Apply(ctx context.Context, updater APIKeyQuotaUpdater) error {
	return s.gateway.ApplyChatBillingReceipt(ctx, s.receipt, updater, func(persist context.Context) error { return s.repo.MarkAttachmentParseBilled(persist, s.record) })
}
