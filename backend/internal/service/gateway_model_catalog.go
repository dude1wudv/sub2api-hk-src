package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/typesafe"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

// GatewayModel retains the complete public entry, including provider extensions.
// The outer ID is for consumers; JSON serialization preserves the original shape.
type GatewayModel struct {
	ID  string
	Raw any
}

func (m GatewayModel) MarshalJSON() ([]byte, error) { return json.Marshal(m.Raw) }

type GatewayModelCatalogService struct {
	gateway            *GatewayService
	openai             *OpenAIGatewayService
	maxAccountSwitches int
}

func NewGatewayModelCatalogService(gateway *GatewayService, openai *OpenAIGatewayService, maxAccountSwitches int) *GatewayModelCatalogService {
	return &GatewayModelCatalogService{gateway: gateway, openai: openai, maxAccountSwitches: maxAccountSwitches}
}

func (s *GatewayModelCatalogService) ListPinnedForGroup(ctx context.Context, group *Group, etag string) (*OpenAIModelsResponse, *Account, error) {
	if s.openai == nil {
		return nil, nil, infraerrors.New(http.StatusInternalServerError, "MODEL_DISCOVERY_UNAVAILABLE", "OpenAI model discovery is not configured")
	}
	return s.openai.FetchPinnedOpenAIModelsList(ctx, group, s.maxAccountSwitches, etag)
}

func (s *GatewayModelCatalogService) ListForGroup(ctx context.Context, group *Group) ([]GatewayModel, error) {
	if s == nil || s.gateway == nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_CATALOG_UNAVAILABLE", "Model catalog is unavailable")
	}
	var groupID *int64
	platform := ""
	if group != nil {
		platform = group.Platform
		if group.ID > 0 {
			groupID = &group.ID
		}
		if platform == PlatformOpenAI && group.CodexModelsManifestConfig.Enabled {
			response, _, err := s.ListPinnedForGroup(ctx, group, "")
			if err != nil {
				return nil, err
			}
			_, entries, err := modelCatalogEntries(response.Body, "data")
			if err != nil {
				return nil, invalidOpenAIModelsList(err)
			}
			models := make([]GatewayModel, 0, len(entries))
			for _, entry := range entries {
				var identity struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(entry, &identity); err != nil || identity.ID == "" {
					return nil, invalidOpenAIModelsList(infraerrors.BadRequest("INVALID_MODEL_ID", "Invalid model catalogue ID"))
				}
				models = append(models, GatewayModel{ID: identity.ID, Raw: entry})
			}
			return models, nil
		}
	}
	var available []string
	if platform == PlatformComposite {
		available = s.CompositeAvailableModels(ctx, groupID, "", true)
		if len(available) == 0 {
			available = DefaultGatewayModelIDs(platform)
		}
		if group != nil && group.ModelAllowlistEnabled() {
			available = group.ModelAllowlist.FilterForListing(available)
		}
		return GatewayModelsForIDs(platform, available), nil
	}
	available = s.gateway.GetAvailableModels(ctx, groupID, platform)
	if group != nil && group.ModelAllowlistEnabled() {
		source := GatewayModelListingSource(platform, available, DefaultGatewayModelIDs(platform))
		return GatewayModelsForIDs(platform, group.ModelAllowlist.FilterForListing(source)), nil
	}
	if len(available) > 0 || (platform == PlatformOpenAI && available != nil) {
		return GatewayModelsForIDs(platform, available), nil
	}
	switch platform {
	case PlatformOpenAI:
		return wrapGatewayModels(openai.DefaultModels, func(m openai.Model) string { return m.ID }), nil
	case PlatformGemini:
		return wrapGatewayModels(geminicli.DefaultModels, func(m geminicli.Model) string { return m.ID }), nil
	case PlatformGrok:
		return GatewayModelsForIDs(platform, xai.DefaultModelIDs()), nil
	case PlatformTypeSafe:
		return GatewayModelsForIDs(platform, []string{typesafe.JevLatestModel}), nil
	default:
		return wrapGatewayModels(claude.DefaultModels, func(m claude.Model) string { return m.ID }), nil
	}
}

func wrapGatewayModels[T any](values []T, id func(T) string) []GatewayModel {
	models := make([]GatewayModel, 0, len(values))
	for _, value := range values {
		models = append(models, GatewayModel{ID: id(value), Raw: value})
	}
	return models
}

func GatewayModelsForIDs(platform string, modelIDs []string) []GatewayModel {
	if platform == PlatformOpenAI {
		return gatewayOpenAIModels(modelIDs)
	}
	if platform == PlatformGrok {
		return gatewayGrokModels(modelIDs)
	}
	models := make([]claude.Model, 0, len(modelIDs))
	for _, id := range modelIDs {
		models = append(models, claude.Model{ID: id, Type: "model", DisplayName: id, CreatedAt: "2024-01-01T00:00:00Z"})
	}
	return wrapGatewayModels(models, func(m claude.Model) string { return m.ID })
}

func (s *GatewayModelCatalogService) CodexModelIDsForGroup(ctx context.Context, group *Group, platformOverride string) []string {
	if s == nil || s.gateway == nil || group == nil {
		return nil
	}

	groupID := &group.ID
	platform := strings.TrimSpace(platformOverride)
	if platform == "" {
		platform = group.Platform
	}
	if platform == PlatformComposite {
		availableModels := s.CompositeAvailableModels(ctx, groupID, CompositeRouteEndpointResponses, false)
		fallbackModels := DefaultCodexModelIDsForPlatform(PlatformComposite)
		models := availableModels
		if len(models) == 0 {
			models = fallbackModels
		}
		if group.ModelAllowlistEnabled() {
			models = group.ModelAllowlist.FilterForListing(models)
		}
		if filtered, err := s.gateway.FilterCompositeCodexModels(ctx, group.ID, models); err == nil {
			return filtered
		}
		return models
	}

	availableModels := s.gateway.GetAvailableModels(ctx, groupID, platform)
	fallbackModels := DefaultCodexModelIDsForPlatform(platform)
	if group.ModelAllowlistEnabled() {
		return group.ModelAllowlist.FilterForListing(GatewayModelListingSource(platform, availableModels, fallbackModels))
	}
	if len(availableModels) > 0 {
		return availableModels
	}
	return fallbackModels
}

// compositeAvailableModels lists the models the composite group can serve.
// includeSystemOne adds TypeSafe models, which only work through /v1/systemone;
// LLM client catalogs (Codex) must exclude them.
func (s *GatewayModelCatalogService) CompositeAvailableModels(ctx context.Context, groupID *int64, endpoint string, includeSystemOne bool) []string {
	if s == nil || s.gateway == nil {
		return nil
	}
	seen := make(map[string]struct{})
	models := make([]string, 0)
	schedulablePlatforms := s.gateway.GetSchedulablePlatforms(ctx, groupID)
	for _, platform := range domain.CompositePrecedencePlatformIDs() {
		if platform == PlatformTypeSafe && !includeSystemOne {
			continue
		}
		platformModels := s.gateway.GetAvailableModels(ctx, groupID, platform)
		if len(platformModels) == 0 && (platform != PlatformOpenAI || platformModels == nil) {
			// CN 供应商没有静态默认模型列表（DefaultGatewayModelIDs 的
			// default 分支是 Claude 列表），composite 下只暴露账号映射键。
			if _, ok := schedulablePlatforms[platform]; ok && !IsMultiProtocolAPIKeyProvider(platform) {
				platformModels = DefaultGatewayModelIDs(platform)
			}
		}
		for _, model := range platformModels {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			if _, ok := seen[model]; ok {
				continue
			}
			seen[model] = struct{}{}
			models = append(models, model)
		}
	}
	// A route can expose a public ID that no account model mapping contains.
	// On lookup failure, retain the existing account-derived catalog only.
	if routeModels, err := s.gateway.GetCompositeRouteModels(ctx, groupID, endpoint, includeSystemOne); err == nil {
		for _, model := range routeModels {
			if _, ok := seen[model]; !ok {
				seen[model] = struct{}{}
				models = append(models, model)
			}
		}
	}
	return models
}

type grokReasoningEffortOption struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	Default bool   `json:"default,omitempty"`
}

type grokModelListItem struct {
	xai.Model
	SupportsReasoningEffort bool                        `json:"supportsReasoningEffort,omitempty"`
	ReasoningEffort         string                      `json:"reasoningEffort,omitempty"`
	ReasoningEfforts        []grokReasoningEffortOption `json:"reasoningEfforts,omitempty"`
}

func gatewayGrokModels(modelIDs []string) []GatewayModel {
	defaults := xai.DefaultModels()
	defaultsByID := make(map[string]xai.Model, len(defaults))
	for _, model := range defaults {
		defaultsByID[model.ID] = model
	}

	models := make([]grokModelListItem, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		model, ok := defaultsByID[modelID]
		if !ok {
			model = xai.Model{
				ID:          modelID,
				Object:      "model",
				OwnedBy:     "xai",
				DisplayName: modelID,
			}
		}
		item := grokModelListItem{Model: model}
		if grokModelSupportsConfigurableReasoning(modelID) {
			item.SupportsReasoningEffort = true
			item.ReasoningEffort = "high"
			efforts := []grokReasoningEffortOption{
				{Value: "low", Label: "Low"},
				{Value: "medium", Label: "Medium"},
				{Value: "high", Label: "High", Default: true},
			}
			if GrokSupportsXHighReasoningEffort(modelID) {
				efforts = append(efforts, grokReasoningEffortOption{Value: "xhigh", Label: "xHigh"})
			}
			item.ReasoningEfforts = efforts
		}
		models = append(models, item)
	}

	return wrapGatewayModels(models, func(m grokModelListItem) string { return m.ID })
}

func grokModelSupportsConfigurableReasoning(modelID string) bool {
	switch strings.ToLower(strings.TrimSpace(modelID)) {
	case "grok-4.7", "grok-4.7-latest", "grok-4.6", "grok-4.6-latest", "grok-4.5", "grok-4.5-latest", "grok", "grok-latest", "grok-build", "grok-build-latest", "grok-build-0.1":
		return true
	default:
		return false
	}
}

func gatewayOpenAIModels(modelIDs []string) []GatewayModel {
	defaultsByID := make(map[string]openai.Model, len(openai.DefaultModels))
	for _, model := range openai.DefaultModels {
		defaultsByID[model.ID] = model
	}

	models := make([]openai.Model, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		if model, ok := defaultsByID[modelID]; ok {
			models = append(models, model)
			continue
		}
		models = append(models, openai.Model{
			ID:          modelID,
			Object:      "model",
			Created:     1704067200,
			OwnedBy:     "openai",
			Type:        "model",
			DisplayName: modelID,
		})
	}
	return wrapGatewayModels(models, func(m openai.Model) string { return m.ID })
}

// GatewayModelListingSource 汇总模型列表过滤的候选来源：账号映射键（availableModels）
// 与平台默认列表（fallbackModels）。账号映射为空时回落默认列表；Anthropic
// 平台两者取并集，其余平台以账号映射键为准。
func GatewayModelListingSource(platform string, availableModels, fallbackModels []string) []string {
	if platform == PlatformOpenAI && availableModels != nil {
		return availableModels
	}
	if len(availableModels) == 0 {
		return fallbackModels
	}
	if platform == PlatformAnthropic {
		return mergeGatewayModelIDs(availableModels, fallbackModels)
	}
	return availableModels
}

func DefaultCodexModelIDsForPlatform(platform string) []string {
	switch platform {
	case PlatformDeepseek:
		return []string{"deepseek-v4-pro", "deepseek-v4-flash", "deepseek-flash"}
	case PlatformMiniMax:
		return []string{"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.5"}
	case PlatformStepFun:
		return []string{"step-3.5-flash", "step-3.5-flash-2603", "step-3.7-flash", "step-5-preview", "step-image-edit-2", "step-router-v1", "stepaudio-2.5-asr", "stepaudio-2.5-chat", "stepaudio-2.5-realtime", "stepaudio-2.5-tts"}
	default:
		return DefaultGatewayModelIDs(platform)
	}
}

func DefaultGatewayModelIDs(platform string) []string {
	switch platform {
	case PlatformOpenAI:
		return openai.DefaultModelIDs()
	case PlatformGemini:
		ids := make([]string, 0, len(geminicli.DefaultModels))
		for _, model := range geminicli.DefaultModels {
			ids = append(ids, model.ID)
		}
		return ids
	case PlatformAntigravity:
		models := antigravity.DefaultModels()
		ids := make([]string, 0, len(models))
		for _, model := range models {
			ids = append(ids, model.ID)
		}
		return ids
	case PlatformAnthropic:
		return claude.DefaultModelIDs()
	case PlatformGrok:
		return xai.DefaultModelIDs()
	case PlatformOpenCodeGo:
		return DefaultOpenCodeGoModelIDs()
	case PlatformMirasim:
		return DefaultMirasimModelIDs()
	case PlatformTypeSafe:
		return []string{"jev-latest"}
	case PlatformComposite:
		ids := make([]string, 0)
		seen := make(map[string]struct{})
		for _, concretePlatform := range domain.CompositePrecedencePlatformIDs() {
			// TypeSafe is deliberately skipped: jev-latest only works through
			// /v1/systemone, so the static fallback never advertises it to LLM
			// clients. compositeAvailableModels lists it when the group can serve it.
			if concretePlatform == PlatformTypeSafe {
				continue
			}
			for _, id := range DefaultGatewayModelIDs(concretePlatform) {
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
		}
		return ids
	default:
		ids := make([]string, 0, len(claude.DefaultModels))
		for _, model := range claude.DefaultModels {
			ids = append(ids, model.ID)
		}
		return ids
	}
}

func mergeGatewayModelIDs(primary, secondary []string) []string {
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	merged := make([]string, 0, len(primary)+len(secondary))
	for _, models := range [][]string{primary, secondary} {
		for _, model := range models {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			if _, ok := seen[model]; ok {
				continue
			}
			seen[model] = struct{}{}
			merged = append(merged, model)
		}
	}
	return merged
}
