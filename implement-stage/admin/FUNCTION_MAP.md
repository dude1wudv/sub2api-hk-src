# 管理端逐页功能对照

依据 `a92691325` 原始源码生成 `FUNCTION_COVERAGE.json`，保留字段 v-model、事件处理器、可见条件和递归组件路径。该文件覆盖 24 个后台路由及自定义页面共 25 个入口、206 个关联文件。自动检查仅证明原 v-model 绑定未减少；行为正确性依靠对应隔离测试和手工核验。

| 原 URL | 新版位置 / 处理 | 原页面文件 |
|---|---|---|
| `/custom/:id` | 管理员自定义菜单继承管理外壳，用户可见菜单仍走用户外壳 | `frontend/src/views/user/CustomPageView.vue` |
| `/admin/dashboard` | 总览：原统计/图表保留，宣传区替换为紧凑工具栏 | `frontend/src/views/admin/DashboardView.vue` |
| `/admin/ops` | 运维：原图表、错误/系统日志、告警、通知及设置；详情改右抽屉 | `frontend/src/views/admin/ops/OpsDashboard.vue` |
| `/admin/audit-logs` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/AuditLogView.vue` |
| `/admin/users` | 用户：原余额/权限/并发/额度/批量与历史；用户编辑分标签 | `frontend/src/views/admin/UsersView.vue` |
| `/admin/groups` | 分组：原计费/平台/路由/Manifest/白名单/复制；四标签编辑 | `frontend/src/views/admin/GroupsView.vue` |
| `/admin/channels/pricing` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/ChannelsView.vue` |
| `/admin/channels/monitor` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/ChannelMonitorView.vue` |
| `/admin/subscriptions` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/SubscriptionsView.vue` |
| `/admin/accounts` | 账号：原表格、排序/筛选/批量/平台工具；汇总可展开；新增详情抽屉与五标签编辑 | `frontend/src/views/admin/AccountsView.vue` |
| `/admin/plugins` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/PluginsView.vue` |
| `/admin/announcements` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/AnnouncementsView.vue` |
| `/admin/proxies` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/ProxiesView.vue` |
| `/admin/redeem` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/RedeemView.vue` |
| `/admin/promo-codes` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/PromoCodesView.vue` |
| `/admin/settings` | 设置：原九类全保留，暖白横向标签，离开时草稿保护 | `frontend/src/views/admin/SettingsView.vue` |
| `/admin/risk-control` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/RiskControlView.vue` |
| `/admin/prompt-audit` | 提示词审计：原只读事件/筛选/详情；事件右抽屉 | `frontend/src/features/prompt-audit/PromptAuditView.vue` |
| `/admin/usage` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/UsageView.vue` |
| `/admin/affiliates/invites` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/affiliates/AdminAffiliateInvitesView.vue` |
| `/admin/affiliates/rebates` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/affiliates/AdminAffiliateRebatesView.vue` |
| `/admin/affiliates/transfers` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/affiliates/AdminAffiliateTransfersView.vue` |
| `/admin/orders/dashboard` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/orders/AdminPaymentDashboardView.vue` |
| `/admin/orders` | 订单：原筛选/退款/详情，详情右抽屉；真实退款不做浏览器验收 | `frontend/src/views/admin/orders/AdminOrdersView.vue` |
| `/admin/orders/plans` | 按工作分类导航进入；全宽暖白列表/表单，原字段、弹窗、条件与请求处理保留 | `frontend/src/views/admin/orders/AdminPaymentPlansView.vue` |

系统设置九类：通用、登录条款、功能开关、安全与认证、用户默认值、网关、支付、邮件、备份。导航可见性仍由原功能开关、简易模式和权限决定；未用合成样稿的简化字段替代现有配置。

账号、分组、用户编辑只是重排展示标签；原输入控件保留挂载。用量页的 Token、缓存、实际/标准费用、上游成本及延迟/TPS 未移除。批量当前页/全部结果、部分失败及重试流程仍在原页面处理器。外部工具仍从侧栏进入，未另建认证链。

关键文件的原始字段和操作明细可在 JSON 的 `files` 中按路径查看；对应源码可直接对比基线。参见 `BUILD_NOTE.md` 的验证证据、合成预览限制和发布分工。
