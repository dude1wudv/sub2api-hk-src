# JP/HK 下游改动保留清单

本文件记录**不能在同步上游时被无意覆盖**的 JP/HK 下游行为。执行 `merge upstream/main`、rebase、批量冲突解决或整文件接受 upstream 版本前，必须逐项核对本清单。

> 原则：以上游实现为基础逐项移植，不得对本清单涉及的文件直接使用整文件 `--theirs`。如果上游已提供等价或更完整实现，可以删除下游补丁，但必须先保留回归测试并记录替代提交和验证结果。

## 合并前后通用检查

1. 合并前保存下游基线：
   - `git log --oneline upstream/main..HEAD`
   - `git diff upstream/main...HEAD -- <记录中的代码与测试文件>`
2. 冲突解决时按“行为不变量”判断，不按文本是否相同判断。
3. 合并后确认代码锚点和回归测试仍存在。
4. 运行记录中列出的定向测试，再运行受影响 package 的完整测试。
5. 若改动影响网关转发，发布后分别验证容器健康、代理入口和真实请求；单元测试不能代替生产验收。

---

## D-20260803-01：GPT-5.6 Responses explicit prompt cache 字段透传

- **状态**：必须保留，除非 upstream 已实现并通过同等测试。
- **引入提交**：`0c11b0edb260c61a5ec80bc433bacdd888cfaa58` (`fix(openai): preserve GPT-5.6 explicit prompt cache`)
- **代码文件**：`backend/internal/service/openai_gateway_forward.go`
- **测试文件**：`backend/internal/service/openai_gateway_service_hotpath_test.go`
- **代码锚点**：`supportsOpenAIExplicitPromptCaching`
- **生产首发镜像**：`sub2api:hk-v0.1.170-explicit-cache-0c11b0edb-20260803`

### 问题背景

非 Codex 客户端进入 OpenAI Responses 转发时，历史兼容逻辑会把 `prompt_cache_options` 当作不受支持字段删除。GPT-5.6 原生 Responses 已使用 explicit prompt caching；删除顶层 `prompt_cache_options` 会使请求中的嵌套 `prompt_cache_breakpoint` 失去启用条件，表现为显式缓存配置经过 Sub2API 后无效。

官方 Codex-family 请求由 `isCodexCLI` 分支绕过这段非 Codex 归一化，因此本补丁不是“全局停止剥离”，而是只扩展 GPT-5.6 原生 OpenAI 路径。

### 必须保持的行为不变量

1. `PlatformOpenAI` 且归一化后的上游模型名以 `gpt-5.6` 开头时：
   - 保留顶层 `prompt_cache_options`；
   - 保留输入项内的 `prompt_cache_breakpoint`；
   - 支持带 provider 前缀的模型名，例如 `openai/gpt-5.6-terra`。
2. GPT-5.5、GPT-5.4 等旧模型继续沿用历史剥离行为，避免向不支持的上游发送字段。
3. Anthropic、Gemini、nil account 以及其他 OpenAI-compatible 平台不得因本补丁被错误放行。
4. 官方 Codex-family 请求原有透传路径不得退化。
5. `prompt_cache_retention` 与 `safety_identifier` 的历史处理不受本补丁影响。

### 上游合并高风险区域

重点检查 `openai_gateway_forward.go` 中以下区域：

- `if !isCodexCLI` 的请求归一化块；
- `unsupportedFields` 的构造与删除循环；
- 模型名重写后使用的 `upstreamModel`；
- `supportsOpenAIExplicitPromptCaching` helper 是否仍按平台和模型双重收窄。

如果 upstream 重构了请求清洗流程，应把上述行为迁移到新的字段能力判断层，而不是机械保留旧 helper 的位置。

### 必须保留的回归覆盖

`openai_gateway_service_hotpath_test.go` 至少应继续覆盖：

- GPT-5.4 删除 `prompt_cache_options`；
- GPT-5.6 保留 `prompt_cache_options.type=explicit`；
- GPT-5.6 保留 `input.0.prompt_cache_breakpoint=true`；
- `openai/gpt-5.6-terra` 前缀归一化；
- Anthropic、nil account、旧模型不放行。

定向验证：

```bash
cd backend
go test ./internal/service -run 'Test(OpenAIGatewayService_Forward_NormalizesMaxTokensAndGatesPromptCacheOptions|SupportsOpenAIExplicitPromptCaching)$' -count=1
go test ./internal/service -count=1
go vet ./internal/service
```

### 2026-08-03 首次部署证据

- HK compose 仅将 `sub2api` 服务切换到上述新镜像；Postgres 与 Redis 未重建。
- 新容器：`running healthy`，`restarts=0`。
- `http://127.0.0.1:8080/health`：HTTP 200，`{"status":"ok"}`。
- Caddy SNI 路径 `https://sub.sunmmyapi.xyz/health`：HTTP 200。
- 部署后的 GPT-5.6 Responses 请求持续返回 HTTP 200，近期日志未发现 panic、fatal、migration/database error。

这组证据确认中继版本已上线且服务稳定；真实 explicit cache 命中率仍应由会发送相关字段的客户端请求和上游 usage 数据单独验收。
