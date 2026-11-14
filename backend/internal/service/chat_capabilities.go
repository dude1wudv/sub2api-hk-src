package service

import (
	"context"
	"slices"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// Configuration eligibility deliberately ignores transient health/quota state.
// Every generation still traverses the gateway's live admission and scheduling.
func (s *ChatModelService) modelCapabilities(ctx context.Context, group *Group, model string, metadata ModelMetadata, policy ChatModelPolicy, candidates []Account) (ChatCapabilities, string, bool, error) {
	caps := ChatCapabilities{Transport: policy.Transport, Reasons: map[string]string{}}
	platform, routingModel := group.Platform, model
	if err := domain.ValidateChatModelPolicies(map[string]ChatModelPolicy{model: policy}); err != nil {
		return caps, platform, false, ErrChatInvalid
	}
	if platform == PlatformComposite {
		decision, err := s.routes.Resolve(ctx, group.ID, model, policy.Transport)
		if err != nil {
			return caps, platform, false, err
		}
		if !decision.Matched || decision.TargetPlatform == PlatformTypeSafe {
			return caps, platform, false, nil
		}
		platform, routingModel = decision.TargetPlatform, decision.UpstreamModel
		ctx = WithCompositeRouteDecision(ctx, decision)
	}
	// The native v1beta ingress admits Gemini groups, plus composite routes to
	// Antigravity. Do not advertise a protocol the actual gateway will reject.
	if policy.Transport == "gemini" && (!IsSafeGeminiModelPathSegment(model) || (platform != PlatformGemini && !(group.Platform == PlatformComposite && platform == PlatformAntigravity))) {
		return caps, platform, false, nil
	}
	compatible := 0
	allNative := true
	allImages := policy.ImageInput || (metadata.MetadataMatch == "exact" && slices.Contains(metadata.InputModalities, "image"))
	for i := range candidates {
		account := &candidates[i]
		if !s.chatAccountCompatible(ctx, account, platform, routingModel) {
			continue
		}
		compatible++
		mapped := account.GetMappedModel(routingModel)
		if !s.isChatModel(mapped) {
			return caps, platform, false, nil
		}
		protocol := ""
		if domain.UsesOpenAIGateway(platform) && account.Type == AccountTypeAPIKey {
			protocol = resolveUpstreamProtocol(account, APIProtocolAnthropic, mapped, nil)
			if s.openai != nil {
				protocol = s.openai.resolveUpstreamProtocolFor(ctx, account, APIProtocolAnthropic, mapped)
			}
		}
		// An explicit policy is still required. Model family or tool_call alone never
		// grants native extensions to a generic or mixed-protocol upstream.
		native := policy.Transport == "messages" && account.Type == AccountTypeAPIKey && protocol == APIProtocolAnthropic && (account.IsDeepseek() || strings.HasPrefix(strings.ToLower(mapped), "deepseek-"))
		allNative = allNative && native
		if !policy.ImageInput && allImages {
			upstream := s.metadata.Resolve(mapped)
			if upstream.MetadataMatch == "exact" && !slices.Contains(upstream.InputModalities, "image") {
				allImages = false
			}
		}
	}
	if compatible == 0 {
		return caps, platform, false, nil
	}
	caps.ImageInput = allImages
	caps.NativeDeepSeekExtensions = policy.NativeDeepSeekExtensions && allNative
	caps.NativeFiles = policy.NativeFiles && allNative
	caps.DocumentInput = policy.DocumentInput && caps.NativeFiles
	caps.NativeSearch = policy.NativeSearch && allNative
	caps.NativeReadURL = policy.NativeReadURL && allNative
	caps.Reasoning = metadata.MetadataMatch == "exact" && metadata.Reasoning != nil && *metadata.Reasoning
	caps.ReasoningEfforts = chatReasoningEfforts(model, caps)
	caps.ToolCall = metadata.MetadataMatch == "exact" && metadata.ToolCall != nil && *metadata.ToolCall
	tools := false
	if caps.ToolCall && (group.Platform == PlatformOpenAI || group.Platform == PlatformDeepseek) {
		for i := range candidates {
			account := &candidates[i]
			if account.Platform == group.Platform && account.Status == StatusActive && account.Schedulable && account.DeepSeekToolsAllowedForGroup(&group.ID) {
				tools = true
				break
			}
		}
	}
	caps.Search = caps.NativeSearch || tools
	caps.ReadURL = caps.NativeReadURL || tools
	for _, item := range []struct {
		key     string
		enabled bool
		reason  string
	}{
		{"image_input", caps.ImageInput, "所选模型或渠道未确认支持图片输入"},
		{"document_input", caps.DocumentInput, "文档需要全部候选渠道支持原生 Messages Files"},
		{"native_deepseek_extensions", caps.NativeDeepSeekExtensions, "未配置原生扩展或候选渠道协议不一致"},
		{"native_files", caps.NativeFiles, "未配置原生 Files 或候选渠道协议不一致"},
		{"native_search", caps.NativeSearch, "未配置兼容的原生搜索渠道"},
		{"native_read_url", caps.NativeReadURL, "未配置兼容的原生链接阅读渠道"},
		{"reasoning", caps.Reasoning, "没有精确模型元数据确认思考能力"},
		{"tool_call", caps.ToolCall, "没有精确模型元数据确认工具调用能力"},
		{"search", caps.Search, "需要原生搜索或同组授权工具与模型工具调用能力"},
		{"read_url", caps.ReadURL, "需要原生链接阅读或同组授权工具与模型工具调用能力"},
		{"voice", false, "语音暂未开放"},
	} {
		if !item.enabled {
			caps.Reasons[item.key] = item.reason
		}
	}
	return caps, platform, true, nil
}

func (s *ChatModelService) chatAccountCompatible(ctx context.Context, account *Account, platform, model string) bool {
	if account == nil || account.Status != StatusActive || !account.Schedulable {
		return false
	}
	if source, ok := CompositeRouteSourceFromContext(ctx); ok && source == CompositeRouteSourceAccount {
		if public, found := RequestedPublicModelFromContext(ctx); found && !explicitModelMappingClaims(*account, public) {
			return false
		}
	}
	if domain.UsesOpenAIGateway(platform) {
		return account.Platform == platform && account.IsModelSupported(model) && account.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityChatCompletions)
	}
	mixed := (platform == PlatformAnthropic || platform == PlatformGemini) && account.Platform == PlatformAntigravity && account.IsMixedSchedulingEnabled()
	if account.Platform != platform && !mixed {
		return false
	}
	if s.gateway != nil {
		return s.gateway.isModelSupportedByAccountWithContext(ctx, account, model)
	}
	return account.IsModelSupported(model)
}

func (s *ChatModelService) isChatModel(model string) bool {
	if s.priceCatalog != nil {
		if price := s.priceCatalog.GetIdentifiedModelPricing(model); price != nil && price.Mode != "" {
			return price.Mode == "chat" || price.Mode == "completion"
		}
	}
	lower := strings.ToLower(model)
	// Known non-chat endpoint families must not sneak in through an account whose
	// empty mapping admits any model. Unknown text aliases remain configurable.
	for _, prefix := range []string{"text-embedding-", "embedding-", "whisper-", "tts-", "dall-e-", "gpt-image-", "sora-"} {
		if strings.HasPrefix(lower, prefix) {
			return false
		}
	}
	return true
}
