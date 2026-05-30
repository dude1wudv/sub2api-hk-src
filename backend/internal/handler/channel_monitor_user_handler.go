package handler

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ChannelMonitorUserHandler 渠道监控用户只读 handler。
type ChannelMonitorUserHandler struct {
	monitorService *service.ChannelMonitorService
	settingService *service.SettingService
	channelService *service.ChannelService
}

// NewChannelMonitorUserHandler 创建 handler。
// settingService 用于每次请求前读取功能开关；关闭时 List/GetStatus 直接返回空/404。
func NewChannelMonitorUserHandler(
	monitorService *service.ChannelMonitorService,
	settingService *service.SettingService,
	channelService *service.ChannelService,
) *ChannelMonitorUserHandler {
	return &ChannelMonitorUserHandler{
		monitorService: monitorService,
		settingService: settingService,
		channelService: channelService,
	}
}

// featureEnabled 返回当前渠道监控功能是否开启。
// settingService 为 nil（测试场景）视为启用。
func (h *ChannelMonitorUserHandler) featureEnabled(c *gin.Context) bool {
	if h.settingService == nil {
		return true
	}
	return h.settingService.GetChannelMonitorRuntime(c.Request.Context()).Enabled
}

// --- Response ---

type channelMonitorUserListItem struct {
	ID                   int64                                `json:"id"`
	Name                 string                               `json:"name"`
	Provider             string                               `json:"provider"`
	GroupName            string                               `json:"group_name"`
	PrimaryModel         string                               `json:"primary_model"`
	PrimaryStatus        string                               `json:"primary_status"`
	PrimaryLatencyMs     *int                                 `json:"primary_latency_ms"`
	PrimaryPingLatencyMs *int                                 `json:"primary_ping_latency_ms"`
	Availability7d       float64                              `json:"availability_7d"`
	ExtraModels          []dto.ChannelMonitorExtraModelStatus `json:"extra_models"`
	Timeline             []channelMonitorUserTimelinePoint    `json:"timeline"`
}

// channelMonitorUserTimelinePoint 主模型最近一次检测的 timeline 点。
// 仅用于用户视图 list 响应，admin 视图不使用。
type channelMonitorUserTimelinePoint struct {
	Status        string `json:"status"`
	LatencyMs     *int   `json:"latency_ms"`
	PingLatencyMs *int   `json:"ping_latency_ms"`
	CheckedAt     string `json:"checked_at"`
}

type channelMonitorUserDetailResponse struct {
	ID        int64                         `json:"id"`
	Name      string                        `json:"name"`
	Provider  string                        `json:"provider"`
	GroupName string                        `json:"group_name"`
	Models    []channelMonitorUserModelStat `json:"models"`
}

type channelMonitorUserModelStat struct {
	Model           string                     `json:"model"`
	LatestStatus    string                     `json:"latest_status"`
	LatestLatencyMs *int                       `json:"latest_latency_ms"`
	Availability7d  float64                    `json:"availability_7d"`
	Availability15d float64                    `json:"availability_15d"`
	Availability30d float64                    `json:"availability_30d"`
	AvgLatency7dMs  *int                       `json:"avg_latency_7d_ms"`
	Pricing         *userSupportedModelPricing `json:"pricing"`
}

func userMonitorViewToItem(v *service.UserMonitorView) channelMonitorUserListItem {
	extras := make([]dto.ChannelMonitorExtraModelStatus, 0, len(v.ExtraModels))
	for _, e := range v.ExtraModels {
		extras = append(extras, dto.ChannelMonitorExtraModelStatus{
			Model:     e.Model,
			Status:    e.Status,
			LatencyMs: e.LatencyMs,
		})
	}
	timeline := make([]channelMonitorUserTimelinePoint, 0, len(v.Timeline))
	for _, p := range v.Timeline {
		timeline = append(timeline, channelMonitorUserTimelinePoint{
			Status:        p.Status,
			LatencyMs:     p.LatencyMs,
			PingLatencyMs: p.PingLatencyMs,
			CheckedAt:     p.CheckedAt.UTC().Format(time.RFC3339),
		})
	}
	return channelMonitorUserListItem{
		ID:                   v.ID,
		Name:                 v.Name,
		Provider:             v.Provider,
		GroupName:            v.GroupName,
		PrimaryModel:         v.PrimaryModel,
		PrimaryStatus:        v.PrimaryStatus,
		PrimaryLatencyMs:     v.PrimaryLatencyMs,
		PrimaryPingLatencyMs: v.PrimaryPingLatencyMs,
		Availability7d:       v.Availability7d,
		ExtraModels:          extras,
		Timeline:             timeline,
	}
}

func userMonitorDetailToResponse(d *service.UserMonitorDetail, pricing map[string]*service.ChannelModelPricing) *channelMonitorUserDetailResponse {
	models := make([]channelMonitorUserModelStat, 0, len(d.Models))
	for _, m := range d.Models {
		models = append(models, channelMonitorUserModelStat{
			Model:           m.Model,
			LatestStatus:    m.LatestStatus,
			LatestLatencyMs: m.LatestLatencyMs,
			Availability7d:  m.Availability7d,
			Availability15d: m.Availability15d,
			Availability30d: m.Availability30d,
			AvgLatency7dMs:  m.AvgLatency7dMs,
			Pricing:         toUserPricing(pricing[m.Model]),
		})
	}
	return &channelMonitorUserDetailResponse{
		ID:        d.ID,
		Name:      d.Name,
		Provider:  d.Provider,
		GroupName: d.GroupName,
		Models:    models,
	}
}

func (h *ChannelMonitorUserHandler) pricingByModel(c *gin.Context, d *service.UserMonitorDetail) map[string]*service.ChannelModelPricing {
	if d == nil {
		return map[string]*service.ChannelModelPricing{}
	}
	if h.channelService == nil || d.Provider == "" || len(d.Models) == 0 {
		return map[string]*service.ChannelModelPricing{}
	}
	channels, err := h.channelService.ListAvailable(c.Request.Context())
	if err != nil {
		return map[string]*service.ChannelModelPricing{}
	}
	return monitorPricingByModelFromAvailableChannels(d, channels)
}

func monitorPricingByModelFromAvailableChannels(
	d *service.UserMonitorDetail,
	channels []service.AvailableChannel,
) map[string]*service.ChannelModelPricing {
	if d == nil || d.Provider == "" || len(d.Models) == 0 {
		return map[string]*service.ChannelModelPricing{}
	}
	out := make(map[string]*service.ChannelModelPricing, len(d.Models))
	modelSet := make(map[string]struct{}, len(d.Models))
	for _, m := range d.Models {
		modelSet[m.Model] = struct{}{}
	}
	for _, ch := range channels {
		if ch.Status != service.StatusActive || !channelMatchesMonitorChannel(ch, d.GroupName, d.Provider) {
			continue
		}
		for _, sm := range ch.SupportedModels {
			if sm.Platform != d.Provider || sm.Pricing == nil {
				continue
			}
			if _, ok := modelSet[sm.Name]; !ok {
				continue
			}
			if _, exists := out[sm.Name]; exists {
				continue
			}
			cp := sm.Pricing.Clone()
			out[sm.Name] = &cp
		}
	}
	return out
}

func channelMatchesMonitorChannel(ch service.AvailableChannel, groupName, platform string) bool {
	if groupName == "" {
		return true
	}
	for _, g := range ch.Groups {
		if g.Platform != platform {
			continue
		}
		if g.Name == groupName {
			return true
		}
	}
	return false
}

// --- Handlers ---

// List GET /api/v1/channel-monitors
func (h *ChannelMonitorUserHandler) List(c *gin.Context) {
	if !h.featureEnabled(c) {
		response.Success(c, gin.H{"items": []channelMonitorUserListItem{}})
		return
	}
	views, err := h.monitorService.ListUserView(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	items := make([]channelMonitorUserListItem, 0, len(views))
	for _, v := range views {
		items = append(items, userMonitorViewToItem(v))
	}
	response.Success(c, gin.H{"items": items})
}

// GetStatus GET /api/v1/channel-monitors/:id/status
func (h *ChannelMonitorUserHandler) GetStatus(c *gin.Context) {
	if !h.featureEnabled(c) {
		response.ErrorFrom(c, service.ErrChannelMonitorNotFound)
		return
	}
	// 复用 admin.ParseChannelMonitorID 保持错误码与日志一致。
	id, ok := admin.ParseChannelMonitorID(c)
	if !ok {
		return
	}
	detail, err := h.monitorService.GetUserDetail(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, userMonitorDetailToResponse(detail, h.pricingByModel(c, detail)))
}
