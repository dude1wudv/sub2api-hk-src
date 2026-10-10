# 用户端与管理端联合发布协调

用户于 2026-10-10 明确要求两侧改动一起提交部署，并授权任务间沟通。

## 用户端主任务

- Task ID: `01a12664-38d4-7192-b09a-da5fb3f4ea7e`（重设计 Sub2API 用户端）。
- 当前基线：`a92691325d9f68cb1ecfdec13a59cad2f9cabb9a`；生产镜像 `sub2api:hk-patrick-key-flow-a92691325d`。
- 正在修改：`views/portal/UsageView.vue`、`components/portal/PortalUsageTable.vue`、`PortalUsageDetail.vue`，新增 `PortalUsageAnalytics.vue` 及用量相关测试、验收材料。可能补充 `frontend/dev/user-preview*` 合成统计数据。
- 管理端独占 App.vue、useAppearance、AdminShell、adminSurface、admin-portal、adminNavigation、管理页面及表单；用户端不改这些文件。
- 两侧完成后由用户端主任务统一验证、提交、推送、HK 构建部署及根仓库指针/发布记录。避免双方同时操作 git index 或生产。
- 此文件是协作状态，双方均可按独立小节补充；不包含秘密。

## 状态与交接

- 用户端：READY FOR INTEGRATION。详细统计/彩色图标已完成，18项定向测试、限定ESLint通过，1440/1000/390px合成UI验收通过；等待管理端READY后共同构建/提交/部署。实现与截图详见 BUILD_NOTE.md。
- 管理端：READY FOR INTEGRATION。源码、定向回归、三档响应式与主题隔离验收已完成；由用户端主任务统一提交部署，交接清单与限制见下方。

## 管理端回复

- 管理端负责人：当前管理端侧任务（Task ID `01a126d2-0042-72a3-b5da-f9d8193e4929`）。
- 2026-10-10：已按用户最新指示确认，由用户端主任务统一提交、推送和生产部署；管理端不操作 Git index、提交、推送或生产。
- 状态：READY。新外壳、工作分类导航、独立暖白主题、详情和编辑及相关回归已完成。
- 文件范围：上述管理端独占文件，新增 `components/admin/AdminFormTabs.vue`、`composables/useAdminFormTabs.ts`；后续涉及 `components/common/BaseDialog.vue` 的管理端可选能力及焦点修复、管理页和对应测试。共享 BaseDialog 的用户端行为将回归。
- 材料：`implement-stage/admin/`；登记 24 个后台路由 + 管理员自定义页，递归 206 个关联文件的原始字段/操作/条件。对应位置见 FUNCTION_MAP.md，逐项清单见 FUNCTION_COVERAGE.json。
- 首轮进度记录：账号创建/编辑、用户编辑、分组创建/编辑的 214 项既有测试通过；最终结果见下方。独立合成预览使用 3420/3421，由 `frontend/dev/admin-preview.mjs` 启动，不改动主任务的 3410/3411 或其 fixtures 文件。
- 已接收联合审查的旧深色偏好问题：`tailwind.config.js` 的 dark 变体已同时排除用户和管理工作区，原 localStorage 偏好不变，相关回归一起执行。
- 管理端代码状态：CODE_READY（2026-10-10）。业务源码已稳定：收敛账号草稿快照为可编辑字段，编辑回填完成后建立基线；新增异步预校验期间的 singleFlight 防重复提交。最终定向 77 文件 / 725 测试通过，vue-tsc 通过，git diff --check 通过。25 个入口 / 206 关联文件的原始字段绑定无缺失。仅继续保存响应式界面证据、核对源码 lint 和整理交接材料；若再改业务源码将明确通知。可开始联合构建，最终发布仍等待下方 READY。
- CODE_READY 后补充：`GroupsView.vue` 仅从创建/编辑的草稿快照各去除 `accountSearchKeyword`（账号搜索词不属于待保存配置）。该文件 ESLint 已通过，分组定向回归补跑中；请将这两行也纳入最终源码 hash，除此之外无新增业务变更。

### 管理端最终交接 — READY（2026-10-10）

- 最终源码清单：`implement-stage/admin/HANDOFF_FILES.json`，36 个前端源码/测试/独立预览文件，15 个管理端材料文件，加本协调文件；另提交清单本身。每项包含 SHA-256。用户端主任务自己的 Portal 改动不在此清单内。原始 `.log` 留在本地，不建议加入提交。
- 最新组回归：`pnpm exec vitest run src/views/admin/__tests__/GroupsView` 4 文件 / 19 项通过；此前错误的精确文件名未命中测试，已用正确前缀完成复验。最终相关回归为 77 文件 / 725 项，另加上述分组 4 文件 / 19 项，均退出 0。首次完整套件的 3 个失败文件也已完成定向复验，未跳过测试或减弱断言。
- 草稿误判已处理：账号快照排除翻译提示、远程开关、能力信息、OAuth 展示信息；保留实际输入、额度、模型映射、认证配置、调度参数。编辑回填未结束不建立基线；分组账号搜索词不计入草稿。真实浏览器中未编辑取消无提示，跨标签修改保留并触发取消确认。
- `vue-tsc --noEmit`、全源码 ESLint、最终 GroupsView ESLint、原字段绑定检查和 `git diff --check` 通过。原工作区全范围 lint 的临时 verify.cjs 既有问题已在 BUILD_NOTE 记录；主任务另行报告可提交文件快照全范围 lint 与完整 build 通过，最后分组两行也已同步并通过 Vite 构建。
- UI：1440/1024/390px；256/56px 导航、手机抽屉、640px 详情、960px/手机全屏编辑、九类设置、总览均留存实际源码截图。验证工作区切换后的标志/滚动与旧 dark 偏好隔离，测试尺寸及临时 HTML class 已恢复。
- 已知限制：预览部分辅助 GET 无合成实现会返回 501；真实上游认证、交易、退款及生产权限组合未作实操作。功能清单是静态绑定证据，不能替代生产验收。详情和测试证据见 BUILD_NOTE.md。
- 本侧无待改业务源码；不操作 Git index、提交、推送、生产或根仓库指针。允许主任务按既有用户授权继续统一提交、CI、指定提交部署及生产验收，无需再次等待本侧确认。
