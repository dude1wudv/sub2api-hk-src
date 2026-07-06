package service

import (
	"context"
	"time"
)

type rateLimitAccountRepoStub struct {
	AccountRepository
	accountsByID           map[int64]*Account
	setErrorCalls          int
	tempCalls              int
	updateCredentialsCalls int
	updateExtraCalls       int
	lastCredentials        map[string]any
	lastExtraUpdates       map[string]any
	lastErrorMsg           string
	lastTempReason         string
	lastErrorID            int64
	lastTempID             int64
}

func (r *rateLimitAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if r.accountsByID != nil {
		if account := r.accountsByID[id]; account != nil {
			return account, nil
		}
	}
	return r.AccountRepository.GetByID(ctx, id)
}

func (r *rateLimitAccountRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	r.setErrorCalls++
	r.lastErrorID = id
	r.lastErrorMsg = errorMsg
	return nil
}

func (r *rateLimitAccountRepoStub) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	r.tempCalls++
	r.lastTempID = id
	r.lastTempReason = reason
	return nil
}

func (r *rateLimitAccountRepoStub) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	r.updateCredentialsCalls++
	r.lastCredentials = shallowCopyMap(credentials)
	return nil
}

func (r *rateLimitAccountRepoStub) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	r.updateExtraCalls++
	r.lastExtraUpdates = shallowCopyMap(updates)
	return nil
}
