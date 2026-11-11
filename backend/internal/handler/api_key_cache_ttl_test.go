//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyCacheTTLRequestContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"inherit", "off", "5m", "1h", "", "2h", "1H"} {
		payload := `{"name":"ttl","anthropic_cache_ttl_mode":"` + mode + `"}`
		for _, target := range []any{&CreateAPIKeyRequest{}, &UpdateAPIKeyRequest{}} {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
			err := c.ShouldBindJSON(target)
			if mode == "" || mode == "2h" || mode == "1H" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
	}
	var create CreateAPIKeyRequest
	var update UpdateAPIKeyRequest
	require.NoError(t, json.Unmarshal([]byte(`{"name":"ttl"}`), &create))
	require.NoError(t, json.Unmarshal([]byte(`{"name":"ttl"}`), &update))
	require.Nil(t, create.AnthropicCacheTTLMode)
	require.Nil(t, update.AnthropicCacheTTLMode)
	for _, mode := range []string{"", "inherit", "off", "5m", "1h"} {
		wire, err := json.Marshal(dto.APIKeyFromService(&service.APIKey{AnthropicCacheTTLMode: mode}))
		require.NoError(t, err)
		var output map[string]any
		require.NoError(t, json.Unmarshal(wire, &output))
		want := mode
		if want == "" {
			want = "inherit"
		}
		require.Equal(t, want, output["anthropic_cache_ttl_mode"])
	}
}
