package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelMetadata describes catalogue evidence only. Prices always come from the
// existing billing resolver; neither family guesses nor this registry set prices.
type ModelMetadata struct {
	Description      string   `json:"description"`
	ContextWindow    *int64   `json:"context_window"`
	MaxOutputTokens  *int64   `json:"max_output_tokens"`
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Reasoning        *bool    `json:"reasoning"`
	ToolCall         *bool    `json:"tool_call"`
	Family           string   `json:"family"`
	ReleaseDate      string   `json:"release_date"`
	MetadataSource   string   `json:"metadata_source"`
	MetadataMatch    string   `json:"metadata_match"`
}

type modelMetadataEntry struct {
	provider string
	model    modelsDevModel
}

// ModelMetadataCatalog uses the existing models.dev schema and six-hour TTL.
// Refreshes run in the background and serve the last successful snapshot on
// failure. Public requests never wait for a registry HTTP request.
type ModelMetadataCatalog struct {
	mu          sync.RWMutex
	entries     map[string][]modelMetadataEntry
	families    []string
	fetchedAt   time.Time
	lastAttempt time.Time
	refreshing  bool
	client      *http.Client
	registryURL string
	now         func() time.Time
}

func NewModelMetadataCatalog() *ModelMetadataCatalog {
	return &ModelMetadataCatalog{client: &http.Client{Timeout: 20 * time.Second}, registryURL: modelsDevRegistryURL, now: time.Now}
}

func unknownModelMetadata() ModelMetadata {
	return ModelMetadata{InputModalities: []string{}, OutputModalities: []string{}, MetadataSource: "none", MetadataMatch: "unknown"}
}

func (c *ModelMetadataCatalog) Resolve(modelID string) ModelMetadata {
	result := unknownModelMetadata()
	if c == nil {
		return result
	}
	c.refreshIfNeeded()
	key := strings.ToLower(strings.TrimSpace(modelID))
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entries := c.entries[key]; len(entries) > 0 {
		preferred := metadataPreferredProvider(key)
		selected := entries[0]
		for _, entry := range entries {
			if entry.provider == preferred {
				selected = entry
				break
			}
		}
		model := selected.model
		result.Description = strings.TrimSpace(model.Description)
		if model.Limit.Context > 0 {
			value := model.Limit.Context
			result.ContextWindow = &value
		}
		if model.Limit.Output > 0 {
			value := model.Limit.Output
			result.MaxOutputTokens = &value
		}
		result.InputModalities = normalizeMetadataModalities(model.Modalities.Input)
		result.OutputModalities = normalizeMetadataModalities(model.Modalities.Output)
		result.Reasoning = copyMetadataBool(model.Reasoning)
		result.ToolCall = copyMetadataBool(model.ToolCall)
		result.Family = strings.TrimSpace(model.Family)
		result.ReleaseDate = strings.TrimSpace(model.ReleaseDate)
		result.MetadataSource, result.MetadataMatch = "models.dev", "exact"
		return result
	}
	// Only the family identity is safe for an unlisted sibling. Limits,
	// capabilities, release dates and prices must not be borrowed from it.
	for _, family := range c.families {
		if key == family || strings.HasPrefix(key, family+"-") {
			result.Family = family
			result.MetadataSource, result.MetadataMatch = "models.dev", "family"
			break
		}
	}
	return result
}

func copyMetadataBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func normalizeMetadataModalities(input []string) []string {
	out := make([]string, 0, len(input))
	seen := map[string]bool{}
	for _, value := range input {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "text", "image", "audio", "video", "pdf":
		default:
			continue
		}
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func metadataPreferredProvider(modelID string) string {
	for _, entry := range []struct{ prefix, provider string }{
		{"claude-", "anthropic"}, {"gpt-", "openai"}, {"o1", "openai"}, {"o3", "openai"}, {"o4", "openai"},
		{"gemini-", "google"}, {"deepseek-", "deepseek"}, {"grok-", "xai"}, {"kimi-", "moonshotai"}, {"mistral-", "mistral"},
	} {
		if strings.HasPrefix(modelID, entry.prefix) {
			return entry.provider
		}
	}
	return ""
}

func (c *ModelMetadataCatalog) refreshIfNeeded() {
	if c.client == nil || c.registryURL == "" {
		return
	}
	now := time.Now()
	if c.now != nil {
		now = c.now()
	}
	c.mu.Lock()
	if c.refreshing || (!c.fetchedAt.IsZero() && now.Sub(c.fetchedAt) < modelsDevRegistryTTL) || (!c.lastAttempt.IsZero() && now.Sub(c.lastAttempt) < 2*time.Minute) {
		c.mu.Unlock()
		return
	}
	c.refreshing, c.lastAttempt = true, now
	c.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		registry, err := c.fetch(ctx)
		var entries map[string][]modelMetadataEntry
		var families []string
		if err == nil {
			entries, families = buildMetadataRegistry(registry)
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		c.refreshing = false
		if err != nil {
			return
		}
		c.entries, c.families = entries, families
		c.fetchedAt = now
	}()
}

func (c *ModelMetadataCatalog) fetch(ctx context.Context) (map[string]modelsDevProvider, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.registryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("model registry HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, upstreamModelsBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > upstreamModelsBodyLimit {
		return nil, fmt.Errorf("model registry exceeds size limit")
	}
	var registry map[string]modelsDevProvider
	if err := json.Unmarshal(body, &registry); err != nil {
		return nil, err
	}
	if len(registry) == 0 {
		return nil, fmt.Errorf("model registry is empty")
	}
	return registry, nil
}

// Used to seed an unpublished catalogue in tests. Production builds a new index
// outside the lock so even a large registry cannot stall catalogue reads.
func (c *ModelMetadataCatalog) installRegistry(registry map[string]modelsDevProvider) {
	c.entries, c.families = buildMetadataRegistry(registry)
}

func buildMetadataRegistry(registry map[string]modelsDevProvider) (map[string][]modelMetadataEntry, []string) {
	entries := map[string][]modelMetadataEntry{}
	families := map[string]bool{}
	providers := make([]string, 0, len(registry))
	for id := range registry {
		providers = append(providers, id)
	}
	sort.Strings(providers)
	for _, providerID := range providers {
		provider := registry[providerID]
		ids := make([]string, 0, len(provider.Models))
		for id := range provider.Models {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			model := provider.Models[id]
			entry := modelMetadataEntry{provider: providerID, model: model}
			key := strings.ToLower(strings.TrimSpace(id))
			if key != "" {
				entries[key] = append(entries[key], entry)
			}
			if alias := strings.ToLower(strings.TrimSpace(model.ID)); alias != "" && alias != key {
				entries[alias] = append(entries[alias], entry)
			}
			if family := strings.ToLower(strings.TrimSpace(model.Family)); family != "" {
				families[family] = true
			}
		}
	}
	familyNames := make([]string, 0, len(families))
	for family := range families {
		familyNames = append(familyNames, family)
	}
	sort.Slice(familyNames, func(i, j int) bool {
		if len(familyNames[i]) != len(familyNames[j]) {
			return len(familyNames[i]) > len(familyNames[j])
		}
		return familyNames[i] < familyNames[j]
	})
	return entries, familyNames
}

// ProvideModelPlazaService preserves the existing constructor for isolated tests.
func ProvideModelPlazaService(channels ChannelRepository, groups GroupRepository, pricing *PricingService, billing *BillingService, resolver *ModelPricingResolver, metadata *ModelMetadataCatalog) *ModelPlazaService {
	service := NewModelPlazaService(channels, groups, pricing, billing, resolver)
	service.metadataCatalog = metadata
	return service
}
