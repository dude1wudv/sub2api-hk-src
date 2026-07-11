# Sub2API JP v0.1.152

相对上一正式版 **v0.1.147**（`8fc93f251`）的源码发布。

- **目标提交**：`8180609b75d14b15d5f078cdb04f23fa21a19abf`
- **分支**：`merge/upstream-v0150-20260710`
- **相对 v0.1.147**：+72 commits
- **上游对齐**：选择性合入 v0.1.150 / post-150 能力，并保留 JP 侧调度、计费与网关定制

---

## 更新说明（摘要）

### OpenAI / GPT-5.6 / Codex
- 合入 GPT-5.6 sol/terra/luna 定价与 256K 长上下文；默认 Codex 走 `terra`
- 透传 GPT-5.6 max effort；修复 Anthropic effort bridge
- 模拟上游缺失的 `cache_write` 计费；解析 `cache_write_tokens` 与 30m TTL
- 首 token 计时覆盖 content/reasoning delta 与 `output_item.added`
- 清理 reasoning summary SSE 空 HTML 注释；剥离 Plus `prompt_cache_options`
- Codex 客户端模型清单（manifest）透传；`response_format` 兼容映射
- post-150：identity / fast user_ids、MCP bridge、cache tokens、ops nil 防护

### Grok
- 官方 Grok 4.5 支持；周配额展示与 OAuth 池卡片
- sticky `previous_response_id` 调度；思考强度与工具桥接
- 视频按秒计费、媒体尺寸清洗；chat completions 上游偏好
- effort 限制为 low/medium/high；模拟 OAuth cache 计费

### 调度 / 网关 / 安全
- 空 `model_mapping` 的 OpenAI OAuth 不再吞全模型；异族厂商前缀黑名单
- Anthropic 无 reset 的 429 进入兜底冷却
- `/v1/messages` 传输层错误对齐 failover；流内 200 SSE 错误写入看板
- compact body-signal SSE bridge；`response.failed` 透传
- 鉴权绕过与站点字段 XSS 修复（`site_name` / `doc_url` / logo URL）
- 末位 admin 降级保护与角色更新

### 管理台 / 用量
- 用量页重排 + 延迟健康列 + 用户 Token 排行
- 账号列表 lifetime TU 徽章；分组已用配额展示
- API Key 最近使用 IP；用户角色可创建/编辑
- Token 激励计数支持分组白名单
- Go 工具链默认 1.26.5

---

## 发布说明

本 tag 标记 JP 私有源码快照，供 HK 生产 `git pull` + 镜像构建部署使用。

**部署提示**
1. 拉取 `merge/upstream-v0150-20260710` 或 tag `v0.1.152`
2. 按现有 runbook 构建镜像并滚动重启（保留 DB/Redis 卷）
3. 冒烟建议：`gpt-5.6-terra`、Grok 4.5 sticky 续写、Codex reasoning 流、管理台用量/账号 TU

**未包含**
- 本次 release 不附带预构建镜像 tarball（与 v0.1.147 不同）；以源码 tag 为准构建
- 未强制同步上游全部 v0.1.151 提交，仅为选择性 port + JP 定制

**完整提交范围**

```text
git log --oneline v0.1.147..v0.1.152
```