package service

import (
	"context"
	"time"
)

type rateLimitAccountRepoStub struct {
	AccountRepository
	setErrorCalls          int
	tempCalls              int
	updateCredentialsCalls int
	lastCredentials        map[string]any
	lastErrorMsg           string
	lastTempReason         string
}

func (r *rateLimitAccountRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	r.setErrorCalls++
	r.lastErrorMsg = errorMsg
	return nil
}

func (r *rateLimitAccountRepoStub) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	r.tempCalls++
	r.lastTempReason = reason
	return nil
}

func (r *rateLimitAccountRepoStub) UpdateCredentials(ctx context.Context, id int64, credentials map[string]any) error {
	r.updateCredentialsCalls++
	r.lastCredentials = cloneCredentials(credentials)
	return nil
}
