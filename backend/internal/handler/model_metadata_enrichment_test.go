//go:build unit

package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelMetadataLookupFunc func(string) service.ModelMetadata

func (f modelMetadataLookupFunc) Resolve(id string) service.ModelMetadata { return f(id) }

func TestModelMetadataEnrichmentPreservesUpstreamAndListMembership(t *testing.T) {
	input := []byte(`{"object":"list","custom_extension":{"keep":true},"data":[{"id":"fixture","description":"upstream wins","context_window":777,"pricing":{"input":1},"custom":42}]}`)
	contextWindow := int64(12345)
	enriched := enrichModelsMetadataWith(input, func(id string) service.ModelMetadata {
		require.Equal(t, "fixture", id)
		return service.ModelMetadata{Description: "registry", ContextWindow: &contextWindow, MetadataMatch: "exact", MetadataSource: "models.dev"}
	})
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(enriched, &decoded))
	models := decoded["data"].([]any)
	require.Len(t, models, 1)
	model := models[0].(map[string]any)
	require.Equal(t, "fixture", model["id"])
	require.Equal(t, "upstream wins", model["description"])
	require.Equal(t, 777.0, model["context_window"])
	require.Equal(t, map[string]any{"input": 1.0}, model["pricing"])
	require.Equal(t, 42.0, model["custom"])
	require.Equal(t, "exact", model["metadata_match"])
	require.Equal(t, map[string]any{"keep": true}, decoded["custom_extension"])
	bad := []byte("not-json")
	require.Equal(t, bad, enrichModelsMetadataWith(bad, func(string) service.ModelMetadata {
		t.Fatal("invalid JSON cannot look up metadata")
		return service.ModelMetadata{}
	}))
}

func TestModelMetadataListRetrieveAndPlazaFlatContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// A nil catalogue is a useful unavailable-registry fixture and never starts HTTP.
	var missing *service.ModelMetadataCatalog
	metadata := missing.Resolve("fixture")
	group := toModelPlazaGroupDTO(&service.PlazaGroup{Models: []service.PlazaModel{{Name: "fixture", ModelMetadata: metadata}}}, nil)
	body, err := json.Marshal(group.Models[0])
	require.NoError(t, err)
	var flat map[string]any
	require.NoError(t, json.Unmarshal(body, &flat))
	require.Equal(t, "unknown", flat["metadata_match"])
	require.Contains(t, flat, "context_window")
	require.Nil(t, flat["context_window"])
	require.Contains(t, flat, "tool_call")
	require.NotContains(t, flat, "metadata", "metadata is flat, not a nested object")
	for _, path := range []string{"/v1/models", "/v1/models/fixture"} {
		recorder := httptest.NewRecorder()
		engine := gin.New()
		write := func(c *gin.Context) {
			c.Set(modelMetadataCatalogContextKey, modelMetadataLookupFunc(func(id string) service.ModelMetadata {
				require.Equal(t, "fixture", id)
				return metadata
			}))
			writeModelsListResponse(c, []gin.H{{"id": "fixture", "custom": true}})
		}
		engine.GET("/v1/models", write)
		engine.GET("/v1/models/:model", write)
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.Contains(t, recorder.Body.String(), `"metadata_match":"unknown"`)
		require.Contains(t, recorder.Body.String(), `"custom":true`)
	}
}

func TestModelMetadataPinnedManifestETagPreserved(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, notModified := range []bool{false, true} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		c.Set(modelMetadataCatalogContextKey, modelMetadataLookupFunc(func(string) service.ModelMetadata {
			t.Fatal("an upstream ETag must still identify exactly the upstream representation")
			return service.ModelMetadata{}
		}))
		body := []byte(`{"object":"list","data":[{"id":"fixture","opaque_field":1}]}`)
		writeOpenAIModelsResponse(c, &service.OpenAIModelsResponse{Body: body, ETag: `"fixture-etag"`, NotModified: notModified})
		require.Equal(t, `"fixture-etag"`, recorder.Header().Get("ETag"))
		if notModified {
			require.Equal(t, http.StatusNotModified, recorder.Code)
			require.Empty(t, recorder.Body.String())
		} else {
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, body, recorder.Body.Bytes())
		}
	}
}
