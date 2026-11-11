//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func metadataRegistryFixture() map[string]modelsDevProvider {
	var registry map[string]modelsDevProvider
	// Synthetic model, limits and date are fixture values, not a product claim.
	_ = json.Unmarshal([]byte(`{"openai":{"models":{"gpt-fixture-a":{"id":"gpt-fixture-a","description":"fixture description","family":"gpt-fixture","release_date":"2020-01-02","reasoning":true,"tool_call":false,"limit":{"context":123456,"output":12345},"modalities":{"input":["text","IMAGE","video","image","unsupported"],"output":["text","audio"]},"cost":{"input":999}}}},"00-reseller":{"models":{"gpt-fixture-a":{"limit":{"context":999999}}}}}`), &registry)
	return registry
}

func seededModelMetadataCatalog() *ModelMetadataCatalog {
	catalog := NewModelMetadataCatalog()
	catalog.installRegistry(metadataRegistryFixture())
	catalog.fetchedAt = time.Now()
	return catalog
}

func TestModelMetadataCatalogExactFamilyUnknown(t *testing.T) {
	catalog := seededModelMetadataCatalog()
	exact := catalog.Resolve("gpt-fixture-a")
	require.Equal(t, "models.dev", exact.MetadataSource)
	require.Equal(t, "exact", exact.MetadataMatch)
	require.Equal(t, int64(123456), *exact.ContextWindow, "first-party provider wins conflicting reseller entry")
	require.Equal(t, int64(12345), *exact.MaxOutputTokens)
	require.Equal(t, []string{"text", "image", "video"}, exact.InputModalities)
	require.Equal(t, []string{"text", "audio"}, exact.OutputModalities)
	require.True(t, *exact.Reasoning)
	require.False(t, *exact.ToolCall, "known false differs from unknown")
	require.Equal(t, "2020-01-02", exact.ReleaseDate)
	require.Equal(t, "fixture description", exact.Description)
	// Returned values must not allow callers to mutate the cached snapshot.
	exact.InputModalities[0] = "changed"
	*exact.ToolCall = true
	again := catalog.Resolve("gpt-fixture-a")
	require.False(t, *again.ToolCall)
	require.Equal(t, "text", again.InputModalities[0])
	family := catalog.Resolve("gpt-fixture-unlisted")
	require.Equal(t, "family", family.MetadataMatch)
	require.Equal(t, "gpt-fixture", family.Family)
	require.Nil(t, family.ContextWindow)
	require.Nil(t, family.MaxOutputTokens)
	require.Nil(t, family.ToolCall)
	require.Nil(t, family.Reasoning)
	require.Empty(t, family.InputModalities)
	body, err := json.Marshal(family)
	require.NoError(t, err)
	require.NotContains(t, string(body), "pricing", "family fallback never borrows prices")
	for _, unknown := range []ModelMetadata{catalog.Resolve("completely-unlisted"), (*ModelMetadataCatalog)(nil).Resolve("gpt-fixture-a")} {
		require.Equal(t, "unknown", unknown.MetadataMatch)
		require.Equal(t, "none", unknown.MetadataSource)
		require.Nil(t, unknown.ContextWindow)
		require.Nil(t, unknown.Reasoning)
	}
}

func TestModelMetadataCatalogBackgroundRefreshStaleAndRetry(t *testing.T) {
	var calls atomic.Int32
	var fail atomic.Bool
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
		}
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		<-release
		require.Empty(t, r.Header.Get("Authorization"), "registry fetch cannot inherit account credentials")
		_ = json.NewEncoder(w).Encode(metadataRegistryFixture())
	}))
	defer server.Close()
	defer unblock()
	catalog := NewModelMetadataCatalog()
	catalog.registryURL = server.URL
	began := time.Now()
	require.Equal(t, "unknown", catalog.Resolve("gpt-fixture-a").MetadataMatch)
	require.Less(t, time.Since(began), time.Second, "hot path must not wait for a blocked registry")
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("refresh did not start")
	}
	for i := 0; i < 20; i++ {
		require.Equal(t, "unknown", catalog.Resolve("gpt-fixture-a").MetadataMatch)
	}
	require.Equal(t, int32(1), calls.Load(), "one background refresh per snapshot")
	unblock()
	require.Eventually(t, func() bool { return catalog.Resolve("gpt-fixture-a").MetadataMatch == "exact" }, 3*time.Second, 10*time.Millisecond)
	fail.Store(true)
	catalog.mu.Lock()
	catalog.fetchedAt = time.Now().Add(-7 * time.Hour)
	catalog.lastAttempt = time.Now().Add(-3 * time.Minute)
	catalog.mu.Unlock()
	require.Equal(t, "exact", catalog.Resolve("gpt-fixture-a").MetadataMatch, "stale snapshot remains available")
	require.Eventually(t, func() bool {
		catalog.mu.RLock()
		defer catalog.mu.RUnlock()
		return calls.Load() == 2 && !catalog.refreshing
	}, 3*time.Second, 10*time.Millisecond)
	for i := 0; i < 20; i++ {
		require.Equal(t, "exact", catalog.Resolve("gpt-fixture-a").MetadataMatch)
	}
	require.Equal(t, int32(2), calls.Load(), "retry delay prevents per-request outage traffic")
}

func TestModelMetadataCatalogPlazaIntegration(t *testing.T) {
	channel := plazaPricedChannel(1, "metadata", []int64{1}, PlatformOpenAI, "gpt-fixture-a", "gpt-fixture-unlisted", "other-unknown")
	service := newPlazaService([]Channel{channel}, []Group{{ID: 1, Name: "g", Platform: PlatformOpenAI}}, nil)
	service.metadataCatalog = seededModelMetadataCatalog()
	groups, err := service.ListGroups(context.Background())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Models, 3)
	byName := map[string]PlazaModel{}
	for _, model := range groups[0].Models {
		byName[model.Name] = model
		require.Equal(t, 3e-6, *model.Pricing.InputPrice, "registry cost cannot replace configured pricing")
	}
	require.Equal(t, "exact", byName["gpt-fixture-a"].MetadataMatch)
	require.Equal(t, "family", byName["gpt-fixture-unlisted"].MetadataMatch)
	require.Equal(t, "unknown", byName["other-unknown"].MetadataMatch)
}

func TestModelsDevMetadataExtendedFieldsAndMerge(t *testing.T) {
	model := metadataRegistryFixture()["openai"].Models["gpt-fixture-a"]
	metadata := upstreamMetadataFromModelsDevModel("gpt-fixture-a", model)
	require.Equal(t, []string{"text", "audio"}, metadata.OutputModalities)
	require.False(t, *metadata.ToolCall)
	require.Equal(t, "gpt-fixture", metadata.Family)
	merged, changed := mergeUpstreamModelMetadata(UpstreamModelMetadata{ID: "gpt-fixture-a"}, metadata)
	require.True(t, changed)
	require.Equal(t, metadata.ToolCall, merged.ToolCall)
	require.Equal(t, metadata.OutputModalities, merged.OutputModalities)
	require.Equal(t, metadata.Family, merged.Family)
	require.Equal(t, metadata.ReleaseDate, merged.ReleaseDate)
}
