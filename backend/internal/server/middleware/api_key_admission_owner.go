// Adapted from ranxi2001/sub2api @ 9620d315, LGPL-3.0-or-later.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// InstallAPIKeyAdmissionOwner captures the original key binding before smart
// routing selects a concrete group. The returned cleanup belongs to auth middleware.
func InstallAPIKeyAdmissionOwner(c *gin.Context, apiKeyService *service.APIKeyService, credential string, trustedClientIP string, key *service.APIKey, googleStyle bool) func() {
	if c == nil || c.Request == nil || apiKeyService == nil || key == nil || (key.ConcurrencyLimit <= 0 && !isResponsesWebSocketRoute(c)) {
		return func() {}
	}
	lookup := apiKeyService.GetByKey
	if isResponsesWebSocketRoute(c) {
		lookup = apiKeyService.GetByKeyUncached
	}
	baseRevalidate := newAPIKeyAuthRevalidator(lookup, credential, trustedClientIP, key)
	originalRouteIDs := append([]int64(nil), key.RoutingGroupIDs...)
	revalidate := func(ctx context.Context) (int, error) {
		limit, err := baseRevalidate(ctx)
		if err != nil || len(originalRouteIDs) == 0 {
			return limit, err
		}
		// Recheck the concrete selected group, not just the raw key's primary group.
		selected, _ := ctx.Value(ctxkey.Group).(*service.Group)
		if selected == nil {
			return limit, nil
		}
		current, err := lookup(ctx, credential)
		if err != nil {
			return 0, err
		}
		candidates, err := apiKeyService.SmartRoutingKeys(ctx, current)
		if err != nil {
			return 0, err
		}
		for _, candidate := range candidates {
			if candidate.GroupID == nil || *candidate.GroupID != selected.ID {
				continue
			}
			permissions := service.APIKeyQueueRequestPermissionsFromContext(ctx)
			if blocked := apiKeyQueueBlockedModel(candidate, permissions); blocked != "" {
				return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.NotFound("MODEL_NOT_ALLOWED", fmt.Sprintf("Model %q is not available for this group", blocked)))
			}
			if err := apiKeyQueueRevokedCapability(candidate, permissions); err != nil {
				return 0, service.NewAPIKeyQueueAuthRejected(err)
			}
			if permission := service.APIKeyQueueImagePermissionFromContext(ctx); permission != nil {
				permission.Set(service.GroupAllowsImageGeneration(candidate.Group))
			}
			return limit, nil
		}
		return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Forbidden("GROUP_NOT_ALLOWED", "Selected route group is no longer available"))
	}
	base := service.WithAPIKeyQueueAuthRevalidator(service.WithAPIKeyQueueRequestPermissions(c.Request.Context(), initialKeyQueuePermissions(c)), revalidate)
	base = service.WithAPIKeyQueueImagePermission(base, service.NewAPIKeyQueueImagePermission())
	ctx, cancel := service.WithAPIKeyAdmissionOwner(base)
	c.Request = c.Request.WithContext(ctx)
	var writer *apiKeyAdmissionWriter
	if key.ConcurrencyLimit > 0 {
		writer = &apiKeyAdmissionWriter{ResponseWriter: c.Writer, ctx: ctx, googleStyle: googleStyle, head: c.Request.Method == http.MethodHead}
		c.Writer = writer
	}
	return func() {
		if writer != nil {
			writer.rejectIfLost()
		}
		cancel()
	}
}

func nextWithAPIKeyAdmissionOwner(c *gin.Context, apiKeyService *service.APIKeyService, credential string, trustedClientIP string, key *service.APIKey, googleStyle bool) {
	cleanup := InstallAPIKeyAdmissionOwner(c, apiKeyService, credential, trustedClientIP, key, googleStyle)
	defer cleanup()
	c.Next()
}

func newAPIKeyAuthRevalidator(lookup func(context.Context, string) (*service.APIKey, error), credential string, trustedClientIP string, initial *service.APIKey) service.APIKeyQueueAuthRevalidator {
	identity := captureAPIKeyQueueIdentity(initial)
	return func(ctx context.Context) (int, error) {
		if lookup == nil || credential == "" || initial == nil {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.ServiceUnavailable("API_KEY_AUTH_UNAVAILABLE", "API key authentication is temporarily unavailable"))
		}
		latest, err := lookup(ctx, credential)
		if err != nil {
			if errors.Is(err, service.ErrAPIKeyNotFound) {
				return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Unauthorized("INVALID_API_KEY", "Invalid API key"))
			}
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.ServiceUnavailable("API_KEY_AUTH_UNAVAILABLE", "API key authentication is temporarily unavailable"))
		}
		if latest == nil || latest.ID != identity.keyID {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Unauthorized("INVALID_API_KEY", "Invalid API key"))
		}
		// disabled / deleted / unknown status are unconditional rejections; an
		// expired or exhausted key is rejected in the queue as well because the
		// wait must not revive a dead authorization.
		if !latest.IsActive() &&
			latest.Status != service.StatusAPIKeyExpired &&
			latest.Status != service.StatusAPIKeyQuotaExhausted {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Unauthorized("API_KEY_DISABLED", "API key is disabled"))
		}
		if latest.Status == service.StatusAPIKeyExpired || latest.IsExpired() {
			return 0, service.NewAPIKeyQueueAuthRejected(service.ErrAPIKeyExpired)
		}
		if latest.Status == service.StatusAPIKeyQuotaExhausted || latest.IsQuotaExhausted() {
			return 0, service.NewAPIKeyQueueAuthRejected(service.ErrAPIKeyQuotaExhausted)
		}
		if latest.User == nil || !latest.User.IsActive() {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Unauthorized("USER_INACTIVE", "User account is not active"))
		}
		if code, message, ok := validateAPIKeyGroupAvailable(latest); !ok && len(latest.RoutingGroupIDs) == 0 {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Forbidden(code, message))
		}
		if !validateAPIKeyGroupAllowed(latest) && len(latest.RoutingGroupIDs) == 0 {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Forbidden("GROUP_NOT_ALLOWED", "API Key 所属专属分组不再允许当前用户使用"))
		}
		// IP ACL: re-run the existing rule check with the trusted IP captured at
		// authentication time. Raw request headers must not be re-parsed here.
		if len(latest.IPWhitelist) > 0 || len(latest.IPBlacklist) > 0 {
			allowed, _ := ip.CheckIPRestrictionWithCompiledRules(trustedClientIP, latest.CompiledIPWhitelist, latest.CompiledIPBlacklist)
			if !allowed {
				return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.Forbidden("ACCESS_DENIED", "Access denied"))
			}
		}
		// A changed key binding, platform or charging mode must not keep the old
		// route and the new authorization mixed; report a retryable configuration
		// change instead of forwarding under stale premises. The reason code is
		// kept for existing callers.
		if apiKeyQueueCoreIdentityChanged(identity, latest) {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.ServiceUnavailable("API_KEY_GROUP_CHANGED", "API key configuration changed; please retry"))
		}
		// Publish the freshest image permission for forwarding gates; a
		// revalidation may have relaxed or revoked it since handshake.
		if len(latest.RoutingGroupIDs) > 0 {
			return latest.ConcurrencyLimit, nil
		}
		if permission := service.APIKeyQueueImagePermissionFromContext(ctx); permission != nil {
			permission.Set(service.GroupAllowsImageGeneration(latest.Group))
		}
		permissions := service.APIKeyQueueRequestPermissionsFromContext(ctx)
		if blocked := apiKeyQueueBlockedModel(latest, permissions); blocked != "" {
			return 0, service.NewAPIKeyQueueAuthRejected(infraerrors.NotFound("MODEL_NOT_ALLOWED", fmt.Sprintf("Model %q is not available for this group", blocked)))
		}
		if capabilityErr := apiKeyQueueRevokedCapability(latest, permissions); capabilityErr != nil {
			return 0, service.NewAPIKeyQueueAuthRejected(capabilityErr)
		}
		return latest.ConcurrencyLimit, nil
	}
}

// apiKeyQueueInitialIdentity is the stable authorization identity captured by
// value when the revalidator is installed. It deliberately excludes everything
// that may legitimately change without affecting this request's permissions.
type apiKeyQueueInitialIdentity struct {
	keyID              int64
	userID             int64
	groupID            *int64
	platform           string
	subscriptionType   string
	claudeCodeOnly     bool
	routeGroupIDs      []int64
	maxReasoningEffort string
	reasoningOverLimit string
	reasoningMappings  []service.ReasoningEffortMapping
}

func captureAPIKeyQueueIdentity(initial *service.APIKey) apiKeyQueueInitialIdentity {
	identity := apiKeyQueueInitialIdentity{}
	if initial == nil {
		return identity
	}
	identity.keyID = initial.ID
	identity.userID = initial.UserID
	identity.routeGroupIDs = append([]int64(nil), initial.RoutingGroupIDs...)
	if initial.GroupID != nil {
		groupID := *initial.GroupID
		identity.groupID = &groupID
	}
	if initial.Group != nil {
		identity.platform = initial.Group.Platform
		identity.subscriptionType = initial.Group.SubscriptionType
		identity.claudeCodeOnly = initial.Group.ClaudeCodeOnly
		identity.maxReasoningEffort = initial.Group.MaxReasoningEffort
		identity.reasoningOverLimit = initial.Group.MaxReasoningEffortOverLimit
		identity.reasoningMappings = append([]service.ReasoningEffortMapping(nil), initial.Group.ReasoningEffortMappings...)
	}
	return identity
}

// apiKeyQueueCoreIdentityChanged reports binding, platform, charging or employee
// usage policy changes that require fresh admission. Names, pricing, routing,
// other permissions and nil/empty collection shapes remain outside this
// identity. A WS session opened before enforcement must re-enter admission.
func apiKeyQueueCoreIdentityChanged(initial apiKeyQueueInitialIdentity, latest *service.APIKey) bool {
	if latest == nil {
		return true
	}
	if initial.userID != 0 && latest.UserID != initial.userID {
		return true
	}
	if (initial.groupID == nil) != (latest.GroupID == nil) {
		return true
	}
	if initial.groupID != nil && *initial.groupID != *latest.GroupID {
		return true
	}
	platform, subscriptionType := "", ""
	claudeCodeOnly := false
	if latest.Group != nil {
		platform = latest.Group.Platform
		subscriptionType = latest.Group.SubscriptionType
		claudeCodeOnly = latest.Group.ClaudeCodeOnly
		if initial.maxReasoningEffort != latest.Group.MaxReasoningEffort || initial.reasoningOverLimit != latest.Group.MaxReasoningEffortOverLimit || !slices.Equal(initial.reasoningMappings, latest.Group.ReasoningEffortMappings) {
			return true
		}
	}
	// A queued request or persistent WS connection must re-enter the public
	// client gate after this policy changes, including off -> on transitions.
	return initial.platform != platform || initial.subscriptionType != subscriptionType || initial.claudeCodeOnly != claudeCodeOnly || !slices.Equal(initial.routeGroupIDs, latest.RoutingGroupIDs)
}

// apiKeyQueueBlockedModel re-checks only the client-written model candidates
// this request may bind, and only when the latest group has the allowlist on.
func apiKeyQueueBlockedModel(latest *service.APIKey, permissions service.APIKeyQueueRequestPermissions) string {
	if latest == nil || latest.Group == nil || !latest.Group.ModelAllowlistEnabled() {
		return ""
	}
	for _, candidate := range permissions.Models {
		if !latest.Group.ModelAllowlist.Allows(candidate) {
			return candidate
		}
	}
	return ""
}

// apiKeyQueueRevokedCapability denies only capabilities the current request
// actually uses. An unrelated permission flip is ignored.
func apiKeyQueueRevokedCapability(latest *service.APIKey, permissions service.APIKeyQueueRequestPermissions) error {
	if latest == nil || latest.Group == nil || permissions.Capabilities == 0 {
		return nil
	}
	group := latest.Group
	if permissions.Capabilities&service.APIKeyQueueCapabilityForbiddenWhenClaudeCodeOnly != 0 && group.ClaudeCodeOnly {
		return infraerrors.Forbidden("CLAUDE_CODE_ONLY", "This group is restricted to Claude Code clients (/v1/messages only)")
	}
	if permissions.Capabilities&service.APIKeyQueueCapabilityMessagesDispatch != 0 && !group.AllowMessagesDispatch {
		return infraerrors.Forbidden("MESSAGES_DISPATCH_NOT_ALLOWED", "This group does not allow /v1/messages dispatch")
	}
	if permissions.Capabilities&service.APIKeyQueueCapabilityLive != 0 && !group.AllowLive {
		return infraerrors.Forbidden("LIVE_NOT_ALLOWED", "Live is not enabled for this group")
	}
	if permissions.Capabilities&service.APIKeyQueueCapabilityImageGeneration != 0 && !service.GroupAllowsImageGeneration(group) {
		return infraerrors.Forbidden("IMAGE_GENERATION_NOT_ALLOWED", service.ImageGenerationPermissionMessage())
	}
	return nil
}

type apiKeyAdmissionWriter struct {
	gin.ResponseWriter
	ctx         context.Context
	googleStyle bool
	head        bool
	rejected    bool
	writeErr    error
}

// Once headers are committed, preserve streaming behavior. Before that point,
// replace the representation entirely; upstream bytes must never follow the 503.
func (w *apiKeyAdmissionWriter) rejectIfLost() bool {
	if w.rejected {
		return true
	}
	if w.Written() || !service.APIKeySlotLeaseLost(w.ctx) {
		return false
	}
	w.rejected = true
	body := `{"type":"error","error":{"type":"api_error","message":"API key concurrency lease lost; please retry later"}}`
	if w.googleStyle {
		body = `{"error":{"code":503,"status":"UNAVAILABLE","message":"API key concurrency lease lost; please retry later"}}`
	}
	headers := w.Header()
	for _, name := range []string{"Content-Encoding", "Content-Range", "Content-Disposition", "ETag", "Last-Modified", "Trailer", "Transfer-Encoding"} {
		headers.Del(name)
	}
	headers.Set("Content-Type", "application/json; charset=utf-8")
	headers.Set("Content-Length", strconv.Itoa(len(body)))
	headers.Set("Cache-Control", "no-store")
	w.ResponseWriter.WriteHeader(http.StatusServiceUnavailable)
	if w.head {
		w.ResponseWriter.WriteHeaderNow()
	} else {
		_, w.writeErr = w.ResponseWriter.WriteString(body)
	}
	return true
}

func (w *apiKeyAdmissionWriter) WriteHeader(status int) {
	if !w.rejectIfLost() {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *apiKeyAdmissionWriter) WriteHeaderNow() {
	if !w.rejectIfLost() {
		w.ResponseWriter.WriteHeaderNow()
	}
}

func (w *apiKeyAdmissionWriter) Write(data []byte) (int, error) {
	if w.rejectIfLost() {
		if w.writeErr != nil {
			return 0, w.writeErr
		}
		return len(data), nil // Consume discarded upstream bytes without mixing bodies.
	}
	return w.ResponseWriter.Write(data)
}

func (w *apiKeyAdmissionWriter) WriteString(data string) (int, error) {
	if w.rejectIfLost() {
		if w.writeErr != nil {
			return 0, w.writeErr
		}
		return len(data), nil
	}
	return w.ResponseWriter.WriteString(data)
}

func (w *apiKeyAdmissionWriter) Flush() { _ = w.FlushError() }
func (w *apiKeyAdmissionWriter) FlushError() error {
	w.rejectIfLost()
	if w.writeErr != nil {
		return w.writeErr
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func initialKeyQueuePermissions(c *gin.Context) service.APIKeyQueueRequestPermissions {
	p := service.APIKeyQueueRequestPermissions{}
	path := c.Request.URL.Path
	if strings.Contains(path, "/live") {
		p.Capabilities |= service.APIKeyQueueCapabilityLive
	}
	if strings.Contains(path, "/images/") {
		p.Capabilities |= service.APIKeyQueueCapabilityImageGeneration
	}
	if strings.Contains(path, "/responses") || strings.Contains(path, "/chat/completions") {
		p.Capabilities |= service.APIKeyQueueCapabilityForbiddenWhenClaudeCodeOnly
	}
	return p
}
