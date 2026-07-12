# Sub2API JP v0.1.153

相对上一正式版 **v0.1.152** 的增量发布。

- **分支**：`merge/upstream-v0150-20260710`
- **发布范围**：Grok OAuth/账号修复、API Key 当前并发展示、Docker 构建可复现性
- **上游策略**：选择性 Port，未合入 Header Override（C-03）及其他大范围上游功能

---

## 更新说明

### Grok OAuth 与账号稳定性
- 对齐 Grok OAuth Responses 客户端身份与 CLI 授权契约
- 重新授权时持久化 CLI 代理配置
- OAuth token 在过期前主动刷新，减少请求中断

### API Key 当前并发
- 用户 Keys 页面新增“当前并发”列
- HTTP 与 OpenAI WebSocket 请求统一追踪 API Key Redis 槽位
- Key 列表按当前页批量读取并发，避免 N+1 查询
- Redis 读取失败时降级为 0，不阻断 Key 管理接口
- 增加中英文 UI 文案、DTO/Wire 接线及页面测试

### 构建与供应链
- 根 Dockerfile 使用 frozen lockfile 和 BuildKit pnpm store cache
- 支持显式传入 npm registry，不再默认改写 registry
- deploy Dockerfile 固定 `pnpm@9.15.9`
- deploy Go 构建启用 `-trimpath`
- 前端构建设置 1536 MiB Node heap 上限

### 安全审计说明
- production audit 当前仅命中 `xlsx@0.18.5` 的两项 high advisory
- npm 源无已修复版本，本次未盲目续期例外、未改依赖或 lockfile
- `xlsx` 替换保留为独立依赖迁移项

---

## 验证

- 后端生产包与 `cmd/server` 构建通过
- handler / DTO 定向测试通过
- Wire 生成物已同步
- 前端 typecheck、KeysView 5 项测试及生产构建通过
- `git diff --check` 通过

## 已知限制

- `go test ./internal/service` 仍被既有测试缺失 `rateLimitAccountRepoStub` 阻塞；生产包构建不受影响
- API Key 删除后的 Redis 槽位仍依赖现有 TTL 自愈，本次未加入主动清理
- `xlsx@0.18.5` advisory 尚未通过依赖替换解决

## 部署

HK 生产使用 GitHub 源码快照构建镜像，仅重建 `sub2api` 应用容器，保留现有 PostgreSQL、Redis、配置和数据卷。部署后检查容器健康状态、本机 `/health` 与公网 HTTPS `/health`。