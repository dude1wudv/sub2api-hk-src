# patrickapi 管理端实现与交接

基线 `a92691325d9f68cb1ecfdec13a59cad2f9cabb9a`。现有工作区内实现，未重置或清理本地改动，未修改后端、依赖、数据库迁移、支付配置或用户端业务页。

## 实现

- 新管理外壳：工作分类导航、搜索、功能开关/简易模式、自定义菜单和外部工具；256px 桌面侧栏、56px 中等窗口图标栏、手机抽屉。原 URL、权限和页面业务保留。
- 暖白主题限定当前管理工作区及 body 上的浮层；保存的旧主题偏好不变。旧 html.dark 不再使管理端或用户端出现深色 Tailwind 样式。管理表格固定紧凑密度。
- 总览撤下大幅宣传区；原统计、图表及快捷操作保留。账号汇总按用户选择默认收起，表格与列偏好、筛选和批量处理保留。
- 新只读账号详情抽屉 640px；运维、订单和提示词事件详情沿用原数据，统一使用抽屉。账号创建/编辑、分组创建/编辑、用户编辑使用最大 960px 的分标签弹窗，手机全屏；平台专属表单仍来自原组件。
- 隐藏标签中的原生校验可定位到对应标签；跨标签输入保留。草稿保护覆盖上述表单与系统设置，账号等待回填完成才记录基线；只比较可编辑字段，不比较远程能力标志、翻译或搜索词，不持久化认证数据。
- 对话框焦点循环/恢复、未保存确认、保存中关闭保护；账号异步预校验及请求期间使用 singleFlight，防止双击重复提交。原财务、权限、批量部分失败、兑换流程未改写。

## 验证记录

| 检查 | 结果 |
|---|---|
| 原始字段清单 | `node implement-stage/admin/inventory.mjs --check`：25 个入口（24 后台 + 管理员自定义页）、206 关联文件，`missingBindings: []` |
| 首轮完整 Vitest | 398 文件中 395 通过；3 个失败文件已逐项修正，未使用跳过或弱化断言 |
| 失败复验 | 6 文件 / 90 项通过：外壳测试替身、翻译键完整性、SparkShadow 外链名称、账号编辑及新交互 |
| 最终定向 Vitest | 77 文件 / 725 项通过；覆盖账号、用户、共享控件、路由外壳、草稿/主题、标签校验、singleFlight 与导航 |
| 分组最终复验 | 4 文件 / 19 项通过（GroupsView.duplicate、compositePlatforms、columnSettings、codexManifest）；补丁仅去除搜索词的草稿误判 |
| TypeScript | `pnpm exec vue-tsc --noEmit` 退出 0 |
| 源码 lint | `pnpm exec eslint src dev/admin-preview.mjs dev/admin-preview-fixtures.mjs tailwind.config.js` 退出 0；最后 GroupsView 两行补丁单独 ESLint 退出 0 |
| 原工作区 `pnpm lint:check` | 被既有 `frontend/tmp/column-menu/verify.cjs` 的 3 个 no-var-requires 阻断；保留原文件，未修改规则或把失败当通过 |
| 联合干净输入 lint / build | 用户端主任务已报告 `pnpm lint:check` 与 `pnpm build`（i18n、vue-tsc、Vite）通过；最后分组两行亦已同步快照，Vite 重构建通过 |
| 差异检查 | `git diff --check` 退出 0，仅行尾规范提示 |

首轮失败原因：管理端外壳替换后测试仍 mock 旧模块；两个新标签使用了不存在的翻译键；外链改为图标后缺少可访问名称。已分别修正测试依赖、使用现有翻译键、保留外链读屏名称。完整套件其他 395 文件的通过结果继续有效，后续只对变更和失败做定向回归。

## 界面证据

使用 `frontend/dev/admin-preview.mjs` 的独立 3420/3421 合成预览，不改动主任务 3410/3411。实际 Vue 源码，未请求生产；截图全部为合成数据。

- `screenshots/accounts-1440.png`：宽屏账号表格、折叠汇总；桌面侧栏实测 256px。
- `screenshots/accounts-1024.png`：图标侧栏实测 56px，页面无横向溢出（宽表自身可滚动）。
- `screenshots/navigation-390.png`：手机导航抽屉；焦点进入抽屉，背景 inert，关闭后返回触发器。
- `screenshots/account-drawer-1440.png`：640px 账号详情及完整原列表背景。
- `screenshots/editor-1440.png`、`editor-390.png`：宽屏标签编辑和手机全屏编辑。
- `screenshots/group-editor-1440.png`：分组模型与路由标签，原平台条件和字段保留。
- `screenshots/settings-1440.png`：九类设置入口与完整通用设置。
- `screenshots/dashboard-1440.png`：运营总览；在 HTML 临时加入 dark 的情况下仍是暖白，测试后恢复原 HTML class。

实际交互核验：未修改账号可以直接关闭；修改备注、切换标签再返回仍保留内容，取消时出现继续编辑/放弃修改；分组取消没有误提示；设置返回用户工作区后 admin 标志移除、用户标志生效、滚动回到顶部；再进入管理端恢复其外壳。浏览器尺寸覆盖已恢复。

## 限制与交接

- 功能清单是原绑定/事件/条件的源码证据，不等同于逐个真实上游或生产交易测试。动态平台、批量部分失败和财务请求沿用原业务，通过既有隔离测试覆盖。
- 合成预览仅覆盖必要读取数据；部分运维/上游探测和设置辅助 GET 返回预览 501，会显示失败提示。未把这些 mock 缺口解释为生产服务失败，也未执行充值、退款、删除生产对象或真实认证。
- 本侧会话禁止子代理，独立假设审阅为 `SWEEP_UNAVAILABLE`；自查和自动化检查不冒充独立审阅。
- `.log` 文件为本地诊断，保留但不纳入建议提交清单。源码、覆盖表和截图的准确范围见 `HANDOFF_FILES.json`。
- 不操作 Git index、提交、推送、生产或根仓库指针。最终构建/CI、指定提交部署、三入口健康及生产 UI 验收由用户端主任务统一完成；本侧不声称已发布。
