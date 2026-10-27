# DeepSeek 原生网页搜索与抓取

使用已获授权的固定分组 API Key，直接向网关发送 DeepSeek 原生 JSON 请求：

```bash
curl https://sub.sunmmyapi.xyz/v1/tools/search \
  -H "Authorization: Bearer $SUB2API_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"DeepSeek API 文档","count":1,"need_content":true}'

curl https://sub.sunmmyapi.xyz/v1/tools/fetch \
  -H "Authorization: Bearer $SUB2API_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://api-docs.deepseek.com/zh-cn/guides/search/"}'
```

请求不需要模型字段。字段及搜索语法见 [DeepSeek 官方文档](https://api-docs.deepseek.com/zh-cn/guides/search/)。网关保留原始 JSON 和上游响应；Fetch 由上游执行抓取。

仅 HTTP 200 且 Search 返回非空结果、Fetch 返回非空正文时结算一次。成功空结果留下零费用量；失败不扣费。单价取分组 `web_search_price_per_call`，乘现有用户专属/分组基础倍率，不叠加 token 高峰倍率。组 223 的既有单价为 USD 0.013，基础倍率 0.4 时为 USD 0.0052/次。用量标识为 `deepseek-search` 和 `deepseek-fetch`。

## 运维配置

在现有 API Key 上游账号 `extra` 中合并配置：

```json
{"deepseek_tools_group_ids":[223]}
```

权限按账号与分组共同限定。空、缺失或非法名单均不开放工具能力；账号仍需属于该组、可调度且为 `openai` 或 `deepseek` 平台的 API Key 账号。共享同一账号的其他分组不会因此开放工具。生产初始配置为组 223 使用账号 1112；现有聊天映射、优先级和其他 `extra` 字段必须保留。更新后复核数据库、调度快照及组内真实调用。

接口只接受固定分组 Key，沿用鉴权、余额、额度、审计和并发控制。首版不自动重试或切换其他上游，工具端点错误不会永久禁用聊天账号。上游错误状态与正文透传，传输超时返回 504，畸形成功响应返回 502。每次入站调用有独立结算 ID；客户端重复请求会作为独立调用处理。

## 发布与回退

按根工作区的 HK 生产规则，从 GitHub 拉取固定提交并在服务器构建，仅切换 `sub2api`。保留共享 PostgreSQL、Redis、网络、数据卷和历史用量；本功能没有数据库迁移。原始生产导出和密钥只留在服务器。验收须包括组 223 真实 Search/Fetch、费用各一笔、组 81 无工具请求发往上游，以及本机/Caddy/公网健康与非目标容器对照。

回退仅恢复原应用镜像和原账号工具白名单字段，不恢复数据库或历史账单。原有 `extra` 应按字段恢复，避免覆盖并行管理操作。
