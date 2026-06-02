package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func shouldExhaustOpenAIStatefulFailover(account *service.Account, statusCode int, previousResponseID, sessionHash string) bool {
	if account == nil {
		return false
	}
	if strings.TrimSpace(previousResponseID) != "" {
		return true
	}
	if account.Type != service.AccountTypeOAuth {
		return false
	}
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusBadGateway:
		return true
	default:
		return false
	}
}
