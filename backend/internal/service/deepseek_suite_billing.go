package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func (s *OpenAIGatewayService) SeedDeepSeekSuiteBilling(ctx context.Context, op *DeepSeekSuiteOperation, in *DeepSeekSuiteInput, input *OpenAIRecordUsageInput) error {
	initial := *input
	initial.Result = suiteResult(op, in)
	if in.Conversation {
		initial.Result.BillingModel = resolveMappedUpstreamModel(input.Account, in.Body, "")
		initial.Result.UpstreamModel = initial.Result.BillingModel
	}
	var err error
	op.Receipt, err = s.PrepareChatAuxiliaryBilling(ctx, &initial)
	if err != nil {
		return err
	}
	repo, err := s.DeepSeekSuiteRepository()
	if err != nil {
		return err
	}
	return repo.SaveDeepSeekOperation(ctx, op)
}

// Persist before charging. Unknown meters remain pending and cannot acquire the
// billing fingerprint of an incorrectly finalized zero-cost operation.
func (s *OpenAIGatewayService) CommitDeepSeekSuite(ctx context.Context, op *DeepSeekSuiteOperation, input *OpenAIRecordUsageInput) error {
	repo, err := s.DeepSeekSuiteRepository()
	if err != nil {
		return err
	}
	op.Receipt, err = s.PrepareChatAuxiliaryBilling(ctx, input)
	if err != nil {
		return err
	}
	op.Status = "settling"
	if input.Result.ChatAuxiliaryUsage != nil {
		for _, item := range input.Result.ChatAuxiliaryUsage.Items {
			if item.MeteringStatus != "confirmed" {
				op.Status = "awaiting_meter"
			}
		}
	}
	if err := repo.SaveDeepSeekOperation(ctx, op); err != nil {
		return err
	}
	if op.Status == "awaiting_meter" {
		return nil
	}
	return s.ApplyChatBillingReceipt(ctx, op.Receipt, input.APIKeyService, func(ctx context.Context) error {
		op.Status = "settled"
		return repo.SaveDeepSeekOperation(ctx, op)
	})
}

func (s *OpenAIGatewayService) StartDeepSeekSuiteRecovery(updater APIKeyQuotaUpdater) {
	if s == nil {
		return
	}
	if _, err := s.DeepSeekSuiteRepository(); err != nil {
		return
	}
	s.deepSeekSuiteOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		s.deepSeekSuiteCancel, s.deepSeekSuiteDone = cancel, make(chan struct{})
		go func() {
			defer close(s.deepSeekSuiteDone)
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				s.recoverDeepSeekSuite(ctx, updater)
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	})
}

func (s *OpenAIGatewayService) recoverDeepSeekSuite(ctx context.Context, updater APIKeyQuotaUpdater) {
	repo, err := s.DeepSeekSuiteRepository()
	if err != nil {
		return
	}
	operations, err := repo.ListDeepSeekPending(ctx)
	if err != nil {
		slog.Warn("deepseek_suite.recovery_query_failed", "error", err)
		return
	}
	for _, op := range operations {
		if ctx.Err() != nil {
			return
		}
		work, cancel := context.WithTimeout(ctx, 30*time.Second)
		if op.Status == "settling" {
			err = s.ApplyChatBillingReceipt(work, op.Receipt, updater, func(ctx context.Context) error {
				op.Status = "settled"
				return repo.SaveDeepSeekOperation(ctx, op)
			})
		} else {
			err = s.recoverDeepSeekFile(work, op)
		}
		cancel()
		if err != nil {
			slog.Warn("deepseek_suite.recovery_pending", "operation_id", op.ID, "error", err)
		}
	}
}

func (s *OpenAIGatewayService) DeepSeekFileAccount(ctx context.Context, key *APIKey, op *DeepSeekSuiteOperation) (*Account, error) {
	return s.DeepSeekSuiteAccount(ctx, key, &Account{ID: op.AccountID}, "files")
}

func (s *OpenAIGatewayService) deepSeekFileRequest(ctx context.Context, account *Account, method, remoteID string, query url.Values) ([]byte, int, error) {
	target, err := s.deepSeekSuiteURL(account, "files", remoteID)
	if err != nil {
		return nil, 0, err
	}
	if query != nil {
		u, _ := url.Parse(target)
		q := u.Query()
		for key, values := range query {
			q[key] = values
		}
		u.RawQuery = q.Encode()
		target = u.String()
	}
	req, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(ctx), method, target, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+account.GetCredential("api_key"))
	req.Header.Set("Accept", "application/json")
	response, err := s.doOpenAIUpstream(req, resolveAccountProxyURL(account), account)
	if err != nil {
		return nil, 0, err
	}
	if response == nil || response.Body == nil {
		return nil, 0, ErrChatStream
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := readChatBounded(response.Body, 8<<20)
	return raw, response.StatusCode, err
}

func (s *OpenAIGatewayService) DeleteDeepSeekFile(ctx context.Context, key *APIKey, op *DeepSeekSuiteOperation) error {
	account, err := s.DeepSeekFileAccount(ctx, key, op)
	if err != nil {
		return err
	}
	_, status, err := s.deepSeekFileRequest(ctx, account, http.MethodDelete, op.RemoteID, nil)
	if err != nil || (status != http.StatusNotFound && (status < 200 || status >= 300)) {
		return errors.New("upstream file deletion failed")
	}
	// Preserve unsettled billing state even if the user deletes the file.
	// The visibility flag is stored independently from settlement.
	op.Metadata = setSuiteFileDeleted(op.Metadata)
	repo, _ := s.DeepSeekSuiteRepository()
	return repo.MarkDeepSeekFileDeleted(ctx, op.UserID, op.GroupID, op.ID)
}

func setSuiteFileDeleted(metadata json.RawMessage) json.RawMessage {
	var m map[string]any
	_ = json.Unmarshal(metadata, &m)
	if m == nil {
		m = map[string]any{}
	}
	m["deleted"] = true
	raw, _ := json.Marshal(m)
	return raw
}

func (s *OpenAIGatewayService) recoverDeepSeekFile(ctx context.Context, op *DeepSeekSuiteOperation) error {
	repo, _ := s.DeepSeekSuiteRepository()
	account, err := s.accountRepo.GetByID(ctx, op.AccountID)
	if err != nil || account == nil || !account.DeepSeekSuiteAllowed(&op.GroupID, "files") {
		return ErrDeepSeekSuiteUnavailable
	}
	if op.UpstreamName == "" {
		return nil // No outbound upload was started.
	}
	var recovered []byte
	if op.RemoteID != "" {
		body, status, fetchErr := s.deepSeekFileRequest(ctx, account, http.MethodGet, op.RemoteID, nil)
		if fetchErr != nil || status != http.StatusOK {
			return errors.New("file metadata remains unavailable")
		}
		recovered = body
	} else {
		after := ""
		for page := 0; page < 100; page++ {
			query := url.Values{"limit": {"100"}}
			if after != "" {
				query.Set("after", after)
			}
			body, status, fetchErr := s.deepSeekFileRequest(ctx, account, http.MethodGet, "", query)
			if fetchErr != nil || status != http.StatusOK {
				return errors.New("upload recovery list unavailable")
			}
			found := gjson.GetBytes(body, "data").Array()
			for _, item := range found {
				if item.Get("filename").String() == op.UpstreamName {
					if recovered != nil {
						return errors.New("ambiguous upstream upload outcome")
					}
					recovered = []byte(item.Raw)
				}
			}
			if recovered != nil || len(found) == 0 || !gjson.GetBytes(body, "has_more").Bool() {
				break
			}
			next := found[len(found)-1].Get("id").String()
			if next == "" || next == after {
				break
			}
			after = next
		}
	}
	if len(recovered) == 0 {
		op.Status = "unknown"
		return repo.SaveDeepSeekOperation(ctx, op)
	}
	op.RemoteID = gjson.GetBytes(recovered, "id").String()
	if !chatValidRemoteID(op.RemoteID) {
		return ErrChatInvalid
	}
	op.Metadata, err = suiteFileMetadata(recovered, op, false)
	if err != nil {
		return err
	}
	op.Status = "awaiting_meter"
	// A missing receipt means the process stopped before authentication/billing
	// snapshots were durably captured. Keep the recovered file available, with
	// a pending billing record for explicit reconciliation; do not re-upload.
	if len(op.Receipt) != 0 {
		receipt, decodeErr := decodeChatBillingReceipt(op.Receipt)
		if decodeErr != nil {
			return decodeErr
		}
		var quantity *int64
		if op.Purpose == "user_data" {
			zero := int64(0)
			quantity = &zero
		} else {
			for _, path := range []string{"usage.pages", "usage.page_count", "page_count", "pages_count"} {
				v := gjson.GetBytes(recovered, path)
				if v.Type == gjson.Number && v.Int() >= 0 && v.Int() <= 1000 && float64(v.Int()) == v.Float() {
					count := v.Int()
					quantity = &count
					break
				}
			}
		}
		if quantity != nil {
			usage, _ := chatAuxiliaryUsage("document_parse", quantity, "confirmed")
			amount, _ := usage.cost()
			receipt.Log.ChatAuxiliaryUsage = usage
			receipt.Cost.TotalCost, receipt.Cost.ActualCost = amount, amount
			receipt.Log.TotalCost, receipt.Log.ActualCost = amount, amount
			receipt.Log.AccountStatsCost = &amount
			post := receipt.post(nil)
			post.RequestPayloadHash = receipt.Command.RequestPayloadHash
			receipt.Command = buildUsageBillingCommand(receipt.Log.RequestID, receipt.Log, post)
			op.Receipt, err = json.Marshal(receipt)
			if err != nil {
				return err
			}
			op.Status = "settling"
		}
	}
	return repo.SaveDeepSeekOperation(ctx, op)
}

func DeepSeekSuiteFileVisible(op *DeepSeekSuiteOperation) bool {
	return op != nil && op.RemoteID != "" && !gjson.GetBytes(op.Metadata, "deleted").Bool() &&
		op.Status != "failed" && op.Status != "deleted"
}

func DeepSeekSuiteFileObject(op *DeepSeekSuiteOperation, anthropic bool) json.RawMessage {
	raw, _ := suiteFileMetadata(op.Metadata, op, anthropic)
	return raw
}

func DeepSeekSuiteOperationID(fileID string) string {
	if !strings.HasPrefix(fileID, "file-ds-") {
		return ""
	}
	return strings.TrimPrefix(fileID, "file-ds-")
}
