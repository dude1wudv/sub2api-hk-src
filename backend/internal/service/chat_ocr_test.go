//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatOCRReceiptRetriesOriginalCommandAfterLogFailure(t *testing.T) {
	logs := &openAIRecordUsageLogRepoStub{inserted: true}
	bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, bills, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	quantity := int64(2)
	usage, err := chatAuxiliaryUsage("ocr", &quantity, "confirmed")
	require.NoError(t, err)
	input := &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "chat_auxiliary:" + uuid.NewString(), Model: "deepseek-ocr", chatAuxiliaryOnly: true, ChatAuxiliaryUsage: usage}, User: &User{ID: 7}, APIKey: &APIKey{ID: 2, UserID: 7, Key: "SYNTHETIC_KEY_MUST_NOT_PERSIST", GroupID: i64p(11), Group: &Group{ID: 11, RateMultiplier: 9}}, Account: &Account{ID: 23, Platform: PlatformOpenAI, Credentials: map[string]any{"api_key": "SYNTHETIC_UPSTREAM_MUST_NOT_PERSIST"}}}
	raw, err := svc.PrepareChatAuxiliaryBilling(t.Context(), input)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "SYNTHETIC_KEY_MUST_NOT_PERSIST")
	require.NotContains(t, string(raw), "SYNTHETIC_UPSTREAM_MUST_NOT_PERSIST")
	require.Zero(t, bills.calls)
	require.Zero(t, logs.calls)
	receipt, err := decodeChatBillingReceipt(raw)
	require.NoError(t, err)
	require.Equal(t, .003, receipt.Command.BalanceCost)
	original, err := json.Marshal(receipt.Command)
	require.NoError(t, err)
	input.APIKey.Group.RateMultiplier = 0
	input.Result.ChatAuxiliaryUsage = nil
	marks := 0
	mark := func(context.Context) error { marks++; return nil }
	bills.err = errors.New("synthetic apply outage")
	require.Error(t, svc.ApplyChatBillingReceipt(t.Context(), raw, nil, mark))
	require.Zero(t, logs.calls)
	require.Zero(t, marks)
	bills.err = nil
	logs.err = errors.New("synthetic usage-row outage")
	require.Error(t, svc.ApplyChatBillingReceipt(t.Context(), raw, nil, mark))
	require.Zero(t, marks)
	bills.result = &UsageBillingApplyResult{Applied: false}
	logs.err = nil
	require.NoError(t, svc.ApplyChatBillingReceipt(t.Context(), raw, nil, mark))
	require.Equal(t, 1, marks)
	applied, err := json.Marshal(bills.lastCmd)
	require.NoError(t, err)
	require.JSONEq(t, string(original), string(applied))
	require.Equal(t, .003, logs.lastLog.ActualCost)
	require.Equal(t, receipt.Command.RequestID, logs.lastLog.RequestID)
}

type chatOCRRepoFake struct {
	ChatRepository
	failed bool
	known  bool
}

func (r *chatOCRRepoFake) FailAttachmentParse(_ context.Context, _ *ChatParseRecord, known bool) error {
	r.failed = true
	r.known = known
	return nil
}

func TestChatOCROnlyAuthoritativePagesAndNoAutomaticReplay(t *testing.T) {
	for _, test := range []struct {
		name, raw string
		status    int
		ok, known bool
	}{
		{"success", `{"id":"ocr-synthetic","object":"ocr.result","purpose":"text_recognition","page_count":1,"pages":[{"index":0,"text":"raw <｜ref｜>text","finish_reason":"length"}],"usage":{"input_tokens":9999}}`, 200, true, false},
		{"missing-pages", `{"id":"ocr-synthetic","object":"ocr.result","purpose":"text_recognition","pages":[]}`, 200, false, false},
		{"provider-error", `{"error":{"message":"OCR did not finish in time"}}`, 200, false, true},
		{"rejected", `{}`, 429, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gateway, key, account, ctx := chatSpeechFixture(t, "ocr")
			_, native, _ := chatNativeFileFixture(t)
			repo := &chatOCRRepoFake{ChatRepository: native.service.repo}
			native.service.repo = repo
			scope := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
			var id uuid.UUID
			for file := range native.files {
				id = file
			}
			scope.chat = native.service
			scope.parse = &ChatParseRecord{UserID: 7, AttachmentID: id, AccountID: account.ID, OperationID: scope.operationID}
			upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: test.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(test.raw))}}
			gateway.httpUpstream = upstream
			result, parsed, err := gateway.ForwardChatOCR(ctx, key, account)
			require.Equal(t, 1, upstream.calls)
			require.Equal(t, http.MethodPost, upstream.request.Method)
			require.True(t, HTTPUpstreamRedirectsDisabled(upstream.request.Context()))
			if test.ok {
				require.NoError(t, err)
				cost, e := result.ChatAuxiliaryUsage.cost()
				require.NoError(t, e)
				require.Equal(t, .0015, cost)
				require.Zero(t, result.Usage.InputTokens)
				require.True(t, strings.Contains(parsed.Pages[0].Text, "<｜ref｜>"))
				require.False(t, repo.failed)
			} else {
				require.Error(t, err)
				require.Nil(t, result)
				require.True(t, repo.failed)
				require.Equal(t, test.known, repo.known)
			}
		})
	}
}

func TestChatDocumentTextDecoding(t *testing.T) {
	for _, raw := range [][]byte{[]byte("hello"), {0xef, 0xbb, 0xbf, 'h', 'e', 'l', 'l', 'o'}, {0xff, 0xfe, 'h', 0, 'e', 0, 'l', 0, 'l', 0, 'o', 0}, {0xfe, 0xff, 0, 'h', 0, 'e', 0, 'l', 0, 'l', 0, 'o'}} {
		text, err := decodeChatDocumentText(raw)
		require.NoError(t, err)
		require.Equal(t, "hello", text)
	}
	for _, raw := range [][]byte{{0xff, 0xfe, 0, 0xd8}, {0xff, 0xfe, 1}, {'a', 0}, {0xff}} {
		_, err := decodeChatDocumentText(raw)
		require.Error(t, err)
	}
}
