# Assumption Ledger — patrickapi 用户入口
<!-- ASK mode: never; 用户要求的重要设计选项优先以异步问题对齐，不阻塞独立工作。 -->

| ID | 未完全确定的选择 | 决定 | Class | Source |
|----|------------------|------|-------|--------|
| A-001 | 首页是聊天还是 API 入口 | API 服务入口，匿名主按钮进入登录；普通用户进入用户控制台；管理员保留原有管理控制台入口，不增加聊天后端 | semantic | user; admin destination default |
| A-002 | 品牌与域名 | 用户界面品牌 patrickapi，公开域名 patrickapi.microedulab.com | interface | user |
| A-003 | 用户与管理共用外壳 | 按路由分离；/admin 与管理员专用自定义页保持旧壳；管理用户也可查看新版个人工作区 | interface | default |
| A-004 | 认证同时服务用户与管理员 | 共享 /login 等认证页换新，但认证逻辑和 /admin 工作区保持不变 | interface | default |
| A-005 | 自定义首页、紧凑首页已有设置 | 保留 home_content 覆盖；compact_home_enabled 使用新版紧凑入口，默认显示完整首页 | interface | default |
| A-006 | 在线模型、价格与用量 | 沿用真实接口和功能开关；不硬编码价格、促销、可用性或使用统计；本地预览仅使用独立合成接口 | semantic | default |
| A-007 | 全屏与非全屏 | 同时验收宽桌面与窄窗口布局，移动端可操作；后台原外观偏好不被强制写入 | interface | user |
| A-008 | 发布权限 | 2026-10-10 用户审阅通过，批准提交、推送和生产发布；包含默认关闭的推理强度功能，先部署后修 CI，仅切换 Sub2API 应用 | interface | explicit user approval |
| A-009 | 素材与品牌 | patrickapi 原创字标；不复制 b.ai 品牌、促销和第三方素材。首轮门户玻璃材质方向已被 A-020 替代 | interface | user revision |
| A-010 | 用户端主题 | 最新要求以 b.ai 实测暖白黑色方案为准，固定单套浅色；/admin 多主题继续保留 | interface | user revision |
| A-011 | 首页玻璃视觉分量 | 最新要求暂不做玻璃，以复刻 b.ai 布局为主；替代此前克制玻璃决定 | interface | user revision |
| A-012 | 用户新手引导 | 用户壳菜单进入新 API 页的四步接入说明；不启动依赖旧密钥模板的遮罩引导；管理员引导仍在原管理壳中启动 | interface | default; rewritten portal contract |
| A-013 | 接入地址显示 | 沿用公开 api_base_url；缺省新公开域名；OpenAI 示例统一 /v1，Anthropic 示例使用 /v1/messages | interface | default |
| A-014 | 窄窗口导航 | 用户确认跟随 b.ai：中等宽度保留图标栏，手机使用抽屉；采用 740/1100px 断点，中等宽度可展开覆盖导航 | interface | user |
| A-015 | 独立支付、回调与自定义内容外观 | 独立用户状态页和弹层也使用固定 b.ai 暖白黑色；Stripe 内嵌组件固定浅色；用户自定义页传 light，管理专用自定义页仍继承管理主题；保留历史回调订单查询和状态轮询，末尾操作改为返回控制台，避免将旧订单指向新的兑换记录 | interface | default; user color revision; sweep |
| A-016 | 用量明细信息密度 | 用户确认保留原 Sub2API Token／费用／延迟三列展示，包含输入输出、缓存量与命中率、输出阶段速率、实际与标准费用、首字/总耗时/输出TPS；沿用延迟健康度阈值及绿/黄/橙/红语义色，缺失数据为中性颜色；仅统一周边材质，不简化数据 | interface | user; sweep |
| A-017 | 用户导航和主操作默认值 | 用户侧栏折叠仅在当前用户壳内保留；余额入口进入新版充值，简易模式隐藏余额/兑换入口 | interface | default; revised for external card purchase |
| A-018 | 管理自定义页冷启动 | 管理员直达自定义页先读取既有管理设置，再选择外壳；加载失败给出重试，不先展示错误的用户内容 | interface | default; audit correction |
| A-019 | 共享暗色工具类隔离 | 用户页面启用 body 表面标记时暂停 Tailwind dark 变体；管理页面原 .dark 匹配及选择器权重不变，不改全局主题偏好或 html 状态 | interface | default; visual verification correction |
| A-020 | 用户否决首版后的实现边界 | 放弃旧卡片页面套材质方式；新建 portal 页面和组件，以 b.ai 布局/交互复刻为目标；只复用统一 API client、Pinia 与无视觉业务工具，认证/支付契约不另造 | interface | user revision; workspace contract |
| A-021 | 主要功能范围 | 本轮 API、用量、充值、订阅、账户；AI 对话、模型切换会话历史和附件能力留到后续 | semantic | user clarification |
| A-022 | b.ai 配色与文字 | 浏览器实测底色 #FFFDF7、文字/主按钮 #000000、边框 #EFEDE3、选中 #F3F0EB、表头 #F7F5EE；正文 system-ui14px，数据页标题 Times New Roman/Times serif22px600 | interface | user request; browser computed styles |
| A-023 | 服务货币与支付能力映射 | 复刻充值/订阅结构，不抄 b.ai 积分倍率、促销或套餐；余额通过发卡网购码及 Sub2API 原有兑换接口增加，商品价格以发卡网为准；历史支付回调契约保留 | semantic | user revision; backend contract |
| A-024 | 新用户导航 | 主导航 API、用量、充值、订阅、账户，其他已有功能归更多；256px平直全高侧栏，中屏56px图标栏，手机抽屉；/dashboard兼容打开新版API入口 | interface | default; b.ai layout mapping |
| A-025 | 账户绑定能力 | 只展示后端已支持的身份提供商与操作，不因参考站包含钱包或其他登录方式而新增虚构绑定能力 | semantic | default; backend contract |
| A-026 | 用量概览与明细筛选 | 概览和曲线按自然月统计全账户；明细筛选只影响请求列表和导出；成功取得曲线后才补齐无请求日期，接口失败不伪装为零用量 | semantic | default |
| A-027 | 首页设计确认 | 用户已确认暖白、大留白、居中字标和 API 接入栏的新首页方向；继续应用至五个主要功能页 | interface | user approval |
| A-028 | 充值与兑换入口 | /purchase 前往 https://catfk.com/shop/UQZYWSZC 购码，回站调用 redeemAPI；/redeem 兼容新版充值；/orders 显示本站兑换记录并链接第三方订单查询；外链无身份参数，新窗口开启 | interface | user; reference product description |
| A-029 | 内置支付开关与发卡网 | payment_enabled 只控制内置支付；外部购码和原兑换能力不依赖该开关；简易模式仍不可兑换，/admin 支付设置不变 | semantic | default; user external payment requirement |
| A-030 | 订阅的购买渠道 | 订阅计划与已拥有权益来自后端；仅在套餐 purchase_mode 允许余额时提供余额订阅，否则展示现有权益和不可购买状态；不为发卡网虚构订阅商品 | semantic | default; backend capability |
| A-031 | 本地兑换交互验收 | 预览只识别 PREVIEW-TOPUP-10 / PREVIEW-TOPUP-50，按合成用户单次使用；余额和历史仅内存变化，重启清空；真实兑换码一律不处理 | interface | default; isolated local verification |
| A-032 | 密钥高级管理 | 在新组件中保留原有额度/限流计数重置与批量编辑能力；批量模式显式开启、当前页选择，部分失败只重试失败项 | interface | sweep; preserve existing capability |
| A-033 | 订阅额度重置时间 | 使用既有 /subscriptions/progress 的权威 resets_at；读取失败显示不可用，不以固定24/168/720小时估算 | semantic | sweep; preserve backend contract |
| A-034 | 智能路由分组顺序 | 已选分组展示首选及编号，允许上下移动；严格按显示顺序提交 routing_group_ids，提示依次回退和实际命中分组计费；批量分组变更不改智能路由密钥，需单独编辑 | semantic | sweep; preserve backend contract |
