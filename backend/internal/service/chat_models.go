package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/google/uuid"
	"slices"
	"strings"
)

type chatGroupAccess interface {
	GetAvailableGroups(context.Context, int64) ([]Group, error)
}
type chatModelCatalog interface {
	ListForGroup(context.Context, *Group) ([]GatewayModel, error)
}
type chatAccountCatalog interface {
	ListModelAvailabilityCandidates(context.Context, *int64, []string, bool) ([]Account, error)
}

type ChatCatalogGroup struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Platform           string  `json:"platform"`
	SubscriptionType   string  `json:"subscription_type"`
	RateMultiplier     float64 `json:"rate_multiplier"`
	UserRateMultiplier float64 `json:"user_rate_multiplier"`
}

type ChatCapabilities struct {
	Transport                string            `json:"transport"`
	ImageInput               bool              `json:"image_input"`
	DocumentInput            bool              `json:"document_input"`
	NativeDeepSeekExtensions bool              `json:"native_deepseek_extensions"`
	NativeFiles              bool              `json:"native_files"`
	NativeSearch             bool              `json:"native_search"`
	NativeReadURL            bool              `json:"native_read_url"`
	Reasoning                bool              `json:"reasoning"`
	ReasoningEfforts         []string          `json:"reasoning_efforts,omitempty"`
	ToolCall                 bool              `json:"tool_call"`
	Search                   bool              `json:"search"`
	ReadURL                  bool              `json:"read_url"`
	Voice                    bool              `json:"voice"`
	Reasons                  map[string]string `json:"reasons"`
}

type ChatCatalogModel struct {
	ID                string               `json:"id"`
	Pricing           *ChannelModelPricing `json:"pricing"`
	Capabilities      ChatCapabilities     `json:"capabilities"`
	Available         bool                 `json:"available"`
	UnavailableReason string               `json:"unavailable_reason"`
	ModelMetadata
}

type ChatCatalog struct {
	Group  ChatCatalogGroup   `json:"group"`
	Models []ChatCatalogModel `json:"models"`
}

type ChatPresetView struct {
	ChatModelPreset
	Group             ChatCatalogGroup `json:"group"`
	Capabilities      ChatCapabilities `json:"capabilities"`
	Available         bool             `json:"available"`
	UnavailableReason string           `json:"unavailable_reason"`
}

type ChatModelService struct {
	repo         ChatPresetRepository
	groups       chatGroupAccess
	catalog      chatModelCatalog
	accounts     chatAccountCatalog
	gateway      *GatewayService
	openai       *OpenAIGatewayService
	priceCatalog *PricingService
	routes       *CompositeRouteResolver
	metadata     *ModelMetadataCatalog
	billing      *BillingService
	pricing      *ModelPricingResolver
}

func NewChatModelService(repo ChatPresetRepository, keys *APIKeyService, catalog *GatewayModelCatalogService, accounts AccountRepository, gateway *GatewayService, openai *OpenAIGatewayService, routes *CompositeRouteResolver, metadata *ModelMetadataCatalog, billing *BillingService, pricing *ModelPricingResolver, priceCatalog *PricingService) *ChatModelService {
	return &ChatModelService{repo: repo, groups: keys, catalog: catalog, accounts: accounts, gateway: gateway, openai: openai, routes: routes, metadata: metadata, billing: billing, pricing: pricing, priceCatalog: priceCatalog}
}

func (s *ChatModelService) Catalog(ctx context.Context, userID, groupID int64, search string) (*ChatCatalog, error) {
	if groupID <= 0 || len(search) > 256 {
		return nil, ErrChatInvalid
	}
	groups, err := s.groups.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].ID == groupID {
			return s.catalogForGroup(ctx, userID, &groups[i], search)
		}
	}
	return nil, ErrChatNotFound
}

func (s *ChatModelService) catalogForGroup(ctx context.Context, userID int64, group *Group, search string) (*ChatCatalog, error) {
	out := &ChatCatalog{Group: ChatCatalogGroup{ID: group.ID, Name: group.Name, Platform: group.Platform, SubscriptionType: group.SubscriptionType, RateMultiplier: group.RateMultiplier, UserRateMultiplier: group.RateMultiplier}, Models: []ChatCatalogModel{}}
	if s.gateway != nil {
		out.Group.UserRateMultiplier = s.gateway.ResolveUserGroupRateMultiplier(ctx, userID, group.ID, group.RateMultiplier)
	}
	if group.ClaudeCodeOnly || group.Platform == PlatformTypeSafe {
		return out, nil
	}
	models, err := s.catalog.ListForGroup(ctx, group)
	if err != nil {
		return nil, err
	}
	platforms := []string{group.Platform}
	if group.Platform == PlatformComposite {
		platforms = domain.CompositePrecedencePlatformIDs()
	} else if group.Platform == PlatformAnthropic || group.Platform == PlatformGemini {
		platforms = append(platforms, PlatformAntigravity)
	}
	candidates, err := s.accounts.ListModelAvailabilityCandidates(ctx, &group.ID, platforms, false)
	if err != nil {
		return nil, err
	}
	search = strings.ToLower(strings.TrimSpace(search))
	for _, model := range models {
		if search != "" && !strings.Contains(strings.ToLower(model.ID), search) {
			continue
		}
		if !group.ModelAllowlist.Allows(model.ID) {
			continue
		}
		metadata := s.metadata.Resolve(model.ID)
		if metadata.MetadataMatch == "exact" && len(metadata.OutputModalities) > 0 && !slices.Contains(metadata.OutputModalities, "text") {
			continue
		}
		if IsImageGenerationIntent("/v1/chat/completions", model.ID, nil) {
			continue
		}
		if !s.isChatModel(model.ID) {
			continue
		}
		policy := ChatModelPolicy{Transport: defaultChatTransport(model.ID)}
		if configured, ok := group.ChatModelPolicies[model.ID]; ok {
			policy = configured
		}
		capabilities, platform, eligible, err := s.modelCapabilities(ctx, group, model.ID, metadata, policy, candidates)
		if err != nil {
			return nil, err
		}
		if !eligible {
			continue
		}
		item := ChatCatalogModel{ID: model.ID, Capabilities: capabilities, Available: true, ModelMetadata: metadata}
		if s.pricing != nil {
			resolved := s.pricing.Resolve(WithResolvedTargetPlatform(ctx, platform), PricingInput{Model: model.ID, GroupID: &group.ID, Group: group})
			if resolved != nil && (resolved.Mode == BillingModeImage || resolved.Mode == BillingModeVideo) {
				continue
			}
			if resolved != nil && resolved.Mode == BillingModePerRequest {
				price := resolved.DefaultPerRequestPrice
				item.Pricing = &ChannelModelPricing{BillingMode: resolved.Mode, PerRequestPrice: &price}
			}
			if s.billing != nil {
				schedule, priceErr := s.billing.ResolveContextPricingSchedule(ctx, s.pricing, ContextPricingScheduleInput{Model: model.ID, Group: group, Platform: platform})
				if priceErr == nil && schedule != nil && len(schedule.Tiers) > 0 {
					item.Pricing = plazaPricingFromSchedule(nil, schedule)
				}
			}
		}
		out.Models = append(out.Models, item)
	}
	return out, nil
}

func (s *ChatModelService) ListPresets(ctx context.Context, userID int64) ([]ChatPresetView, error) {
	presets, err := s.repo.ListPresets(ctx, userID)
	if err != nil {
		return nil, err
	}
	groups, err := s.groups.GetAvailableGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	byGroup := make(map[int64]*Group, len(groups))
	for i := range groups {
		byGroup[groups[i].ID] = &groups[i]
	}
	catalogs := make(map[int64]*ChatCatalog)
	out := make([]ChatPresetView, 0, len(presets))
	for _, preset := range presets {
		view := ChatPresetView{ChatModelPreset: preset, Group: ChatCatalogGroup{ID: preset.GroupID}, UnavailableReason: "分组已停用或不再有访问权限"}
		group := byGroup[preset.GroupID]
		if group != nil {
			catalog, loaded := catalogs[group.ID]
			if !loaded {
				catalog, _ = s.catalogForGroup(ctx, userID, group, "")
				catalogs[group.ID] = catalog
			}
			view.Group.Name = group.Name
			view.Group.Platform = group.Platform
			view.Group.SubscriptionType = group.SubscriptionType
			view.UnavailableReason = "模型目录暂时不可用"
			if catalog != nil {
				view.Group = catalog.Group
				view.UnavailableReason = "模型已下架或没有兼容的聊天渠道"
				for _, model := range catalog.Models {
					if model.ID == preset.Model {
						view.Available = model.Available
						view.UnavailableReason = model.UnavailableReason
						view.Capabilities = model.Capabilities
						break
					}
				}
			}
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *ChatModelService) ResolvePreset(ctx context.Context, userID int64, id uuid.UUID) (*ChatModelPreset, *ChatCatalogModel, *ChatCatalogGroup, error) {
	preset, err := s.repo.GetPreset(ctx, userID, id)
	if err != nil {
		return nil, nil, nil, err
	}
	model, group, err := s.resolveModel(ctx, userID, preset.GroupID, preset.Model)
	return preset, model, group, err
}

func (s *ChatModelService) resolveModel(ctx context.Context, userID, groupID int64, model string) (*ChatCatalogModel, *ChatCatalogGroup, error) {
	catalog, err := s.Catalog(ctx, userID, groupID, model)
	if err != nil {
		return nil, nil, err
	}
	for i := range catalog.Models {
		entry := &catalog.Models[i]
		if entry.ID == model && entry.Available {
			return entry, &catalog.Group, nil
		}
	}
	return nil, nil, ErrChatNotFound
}

func (s *ChatModelService) CreatePreset(ctx context.Context, userID, groupID int64, model, name string) (*ChatModelPreset, error) {
	if err := validateChatPreset(groupID, model, name); err != nil {
		return nil, err
	}
	if _, _, err := s.resolveModel(ctx, userID, groupID, model); err != nil {
		return nil, err
	}
	return s.repo.CreatePreset(ctx, userID, groupID, model, name)
}

func (s *ChatModelService) UpdatePreset(ctx context.Context, userID int64, id uuid.UUID, patch ChatPresetPatch) (*ChatModelPreset, error) {
	for _, effort := range []*string{patch.NormalReasoningEffort, patch.ThinkingReasoningEffort} {
		if effort != nil && !validChatReasoningEffort(*effort) {
			return nil, ErrChatInvalid
		}
	}
	current, err := s.repo.GetPreset(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if patch.GroupID != nil {
		current.GroupID = *patch.GroupID
	}
	if patch.Model != nil {
		current.Model = *patch.Model
	}
	if patch.DisplayName != nil {
		current.DisplayName = *patch.DisplayName
	}
	if err := validateChatPreset(current.GroupID, current.Model, current.DisplayName); err != nil {
		return nil, err
	}
	bindingChanged := patch.GroupID != nil || patch.Model != nil || (patch.IsDefault != nil && *patch.IsDefault)
	effortChanged := patch.NormalReasoningEffort != nil || patch.ThinkingReasoningEffort != nil
	if bindingChanged || effortChanged {
		model, _, err := s.resolveModel(ctx, userID, current.GroupID, current.Model)
		if err != nil {
			return nil, err
		}
		for _, effort := range []*string{patch.NormalReasoningEffort, patch.ThinkingReasoningEffort} {
			if effort != nil && !slices.Contains(chatReasoningEfforts(current.Model, model.Capabilities), *effort) {
				return nil, ErrChatCapability
			}
		}
		if bindingChanged {
			// Persist the exact pair validated above, including a partially changed binding.
			patch.GroupID = &current.GroupID
			patch.Model = &current.Model
		}
	}
	return s.repo.UpdatePreset(ctx, userID, id, patch)
}

func (s *ChatModelService) OrderPresets(ctx context.Context, userID int64, ids []uuid.UUID) error {
	return s.repo.OrderPresets(ctx, userID, ids)
}
func (s *ChatModelService) DeletePreset(ctx context.Context, userID int64, id uuid.UUID) error {
	return s.repo.DeletePreset(ctx, userID, id)
}
