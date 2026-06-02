package handler

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestShouldExhaustOpenAIStatefulFailover(t *testing.T) {
	oauth := &service.Account{Type: service.AccountTypeOAuth}
	apiKey := &service.Account{Type: service.AccountTypeAPIKey}

	require.True(t, shouldExhaustOpenAIStatefulFailover(apiKey, http.StatusTooManyRequests, "resp_1", ""))
	require.True(t, shouldExhaustOpenAIStatefulFailover(oauth, http.StatusUnauthorized, "", "session"))
	require.True(t, shouldExhaustOpenAIStatefulFailover(oauth, http.StatusForbidden, "", ""))
	require.True(t, shouldExhaustOpenAIStatefulFailover(oauth, http.StatusBadGateway, "", "session"))

	require.False(t, shouldExhaustOpenAIStatefulFailover(apiKey, http.StatusTooManyRequests, "", "session"))
	require.False(t, shouldExhaustOpenAIStatefulFailover(apiKey, http.StatusUnauthorized, "", ""))
	require.False(t, shouldExhaustOpenAIStatefulFailover(apiKey, http.StatusBadGateway, "", "session"))
	require.False(t, shouldExhaustOpenAIStatefulFailover(nil, http.StatusUnauthorized, "resp_1", "session"))
}
