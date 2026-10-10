# Build Note — patrickapi 用户入口（b.ai 复刻版）

本记录只将第二轮 b.ai 复刻版作为当前验收对象。首轮淡蓝玻璃方案已被用户否决；其 381 files / 2,926 tests 及首轮截图不作为本轮完成证据。

| Rung | 功能 | 验收 | 状态 |
| --- | --- | --- | --- |
| F0 | 隔离运行入口 | `node dev/user-preview.mjs`，127.0.0.1:3410，合成 API 3411 | 已运行 |
| F1 | 已获用户确认的首页、暖白黑色用户壳和三档导航 | 首页/用户壳/管理隔离回归；1440 / 860 / 390px 浏览器验收 | 通过 |
| F2 | 新建 API、模型价格与用量页面 | 密钥 CRUD、批量编辑、路由顺序；详细 Token/费用/延迟；筛选、CSV、自然月汇总 | 通过 |
| F3 | 发卡网购码、本站兑换、订阅与账户 | 合成兑换成功/重复拦截；订阅余额购买幂等；后端额度重置时间；账户与认证沿用原契约 | 通过本地验证，真实交易未执行 |
| F4 | 构建、回归及管理边界 | `pnpm build`、`pnpm test:run`、`pnpm lint:check`；/admin 原外壳 | 构建/回归通过；完整 lint 有既有临时文件错误 |

## 当前验证记录

- 最终第二轮定向测试：18 files / 126 tests，exit 0。
- `pnpm test:run` 最终 exit 0：392 files / 2,965 tests。首跑发现既有文档链接安全测试引用旧 store 别名，适配到当前别名并保留 sanitizeUrl 断言后全量重跑通过。
- `pnpm build` exit 0：check:i18n 的 3 项检查、vue-tsc、Vite 构建均通过。
- `pnpm lint:check` exit 1：仅既有 `tmp/column-menu/verify.cjs` 的 3 项 no-var-requires。
- 源码范围 `pnpm exec eslint src dev tailwind.config.js --ext '.vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts' --ignore-pattern 'tmp/**'` exit 0。这是业务源码范围证据，不等同完整 lint 通过。使用 `eslint .` 并追加临时目录排除的两次尝试因 Windows CLI 文件匹配失败，没有作为通过证据。
- 主线最终 `git diff --check` exit 0。
- API 批量编辑定向 ESLint、`vue-tsc --noEmit`、`git diff --check`：exit 0。
- 本地预览脚本两个文件 `node --check`：exit 0。
- 浏览器：新登录成功，创建/复制/停用合成密钥；用量筛选空态及还原、请求详情与 Escape 关闭。
- 浏览器：`PREVIEW-TOPUP-10` 将合成余额 128.64 增至 138.64；重复兑换被拦截，余额不再增加；历史记录出现并掩码。服务重启恢复初始数据。
- 布局：1440px 全侧栏，860px 侧栏宽 56px，390px 手机抽屉；手机页面宽度未溢出，明细表在容器内横向滚动，导航后抽屉关闭。
- 管理隔离：/admin 浅色背景 rgb(226,236,246)、暗色 rgb(10,21,35)，无用户表面标记；在 html 保持 dark 时切入用户页，背景仍为 rgb(255,253,247)，返回 /admin 仍为暗色。验收后恢复原浅色偏好与普通合成用户。
- 订阅：浏览器确认当前权益的三个用量/额度和后端 resets_at 倒计时；购买确认显示当前余额、扣除金额及账户余额支付方式，未提交扣款。批量密钥弹窗按选择显示字段与路由限制。
- 接口审查修正：自然月用明确日期，不使用后端滚动月 period；订阅从真实列表包装结构解析 resets_at；密钥重置和批量能力补齐，智能路由顺序可见、可排序、可移除失效项；延迟沿用原业务阈值。
- 已知历史 lint 限制：`frontend/tmp/column-menu/verify.cjs` 是既有、被 Git 忽略的临时文件，有 3 项 no-var-requires。保留完整命令失败记录，没有修改该文件或弱化项目配置。

## 预览与证据

运行与演示账号见 `frontend/dev/README.md`。截图保存在 `implement-stage/screenshots/bai-*.jpg`；其中新充值截图体现用户最后确认的发卡网兑换流程。所有金额、模型、密钥和账户均为合成示例。

预览不连接生产、数据库或模型上游，不执行真实购码、支付、订阅扣款、注册、密码修改、OAuth 或外链提交。真实财务/认证联调与部署验收未执行，不将 mock 检查视为生产证明。

## 授权与后续范围

- /admin 改版、AI 对话按用户要求留到后续。
- 上述记录为发布前本地验收。2026-10-10 用户审阅通过并批准提交、推送与生产发布，明确先部署后修 CI；实际部署 SHA、健康与迁移证据另见根仓库发布报告。
- 语义假设见 ASSUMPTIONS.md；审查以同家族 provisional 记录，不宣称跨模型独立通过。

## 发布后密钥界面修订（2026-10-10）

用户在生产发布后要求 API 页首屏以密钥管理为主，创建时再选分组与配置；分组选择和智能路由参考原 Sub2API 排布，保留分组描述、倍率及既有功能。

- API 首屏移除分组/模型表；模型与完整定价保留独立页，首页模型搜索和旧模型查询链接继续转交目录。调用示例按需展开。
- 创建/编辑使用两栏弹窗：左侧搜索名称或描述、供应商分类与分组卡片；右侧所选摘要、名称、有效期、额度。卡片及摘要显示分组描述；读取 `/groups/rates` 显示用户专属倍率，保留高峰说明，不新增折算“实际倍率”。
- 智能路由沿用固定/智能模式、搜索添加、最多10、有序回退、首选标记、上下移动和失效候选移除；高级设置保留 IP 白黑名单、自定义密钥、三个窗口限额和编辑时显式重置。未新增后端不支持的密钥级模型白名单。
- 定向 7 文件 / 46 测试通过；新增 ModelsView 异步搜索匹配测试 1/1 通过。`pnpm build`（i18n 3 项、vue-tsc、Vite）、修改源码和测试范围 ESLint、diff 检查均 exit 0。没有重复跑全量测试；构建保留既有 chunk/dynamic import/Browserslist 警告。
- 浏览器验证 1440px 两栏及 390px 单栏，手机页面 scrollWidth=390，弹窗宽370，底部操作区在可视范围内。合成智能路由密钥创建成功，重开编辑后 OpenAI → Claude 候选顺序保持。
- 本轮修订截图：`key-workspace-refinement.png`、`key-create-refinement.png`、`key-routing-refinement.png`、`key-create-mobile-refinement.png`，均为本地合成预览。2026-10-10 用户已明确批准“提交部署”；实际固定提交及生产结果记录于根仓库发布报告。
- 上一版生产运行 `32160dbe733032ff52eb6814919bde4dda45accd`；发布后 CI 修复 `f61088a64967d0bc431664ac3342cd08db722d86` 的 CI 与安全扫描已全部通过，生产没有因 CI 清理再次重启。完整发布记录见根仓库 `docs/ops/reports/2026-10-10-patrickapi-portal-release.md`。
