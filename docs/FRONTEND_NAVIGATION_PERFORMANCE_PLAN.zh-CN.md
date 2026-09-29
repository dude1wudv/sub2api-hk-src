# 前端页面切换性能优化计划

## 背景

后台切换侧边栏页面时，会先出现一段空白，然后才渲染出页面，体感较卡。原因有两个：

1. 每个页面都在自己的模板里套一层 `<AppLayout>`，所以每次路由切换都会销毁并重建整个外壳（侧边栏、顶栏、背景、玻璃交互监听）。
2. 页面是懒加载的，点击后才开始下载和解析代码。其中 `GroupsView`、`SettingsView`、`AccountsView` 等页面的 chunk 达到数百 KB。

## 已完成（阶段一）

### 1. 常驻外壳

- `AppLayout.vue` 更名为 `AppShell.vue`，由 `App.vue` 的 `<RouterView v-slot>` 统一包裹。页面组件不再自己套布局。
- 路由通过 `meta.appLayout` 声明是否使用外壳。这个字段可以是布尔值，也可以是 `(route) => boolean`，用于按 query 切换形态的页面：
  - `/admin/ops?fullscreen=1`：运维全屏
  - `/payment/stripe?method=...`：支付弹窗
  - `/model-plaza?embedded=1`：已登录用户的内嵌广场
- 外壳常驻后，原本靠组件重新挂载来重置的状态改为显式处理：导航后清空侧边栏检索词、收起顶栏用户菜单。
- 新增页面时，如果需要后台布局，必须在路由上加 `appLayout: true`。不要在页面里再套外壳，否则会出现两层外壳。

### 2. 页面代码预加载

- 鼠标悬停或键盘聚焦侧边栏链接时，立即预加载目标页面（`prefetchRoutePath`）。触屏设备不触发。
- 登录进入工作区后，利用浏览器空闲时间逐个预热同一工作区的所有 `appLayout` 页面（`scheduleWorkspaceWarmup`）：
  - 普通用户不会预加载管理员页面。
  - 省流量模式或 2G 网络下跳过。
  - 登出后停止剩余的预热队列。

## 待办（阶段二，未实施）

### 3. 常用列表页缓存（KeepAlive）

目标：切回分组、账号、用户、用量等列表页时立即显示上次的内容，同时在后台静默刷新。

方案：

- 在 `AppShell` 内容区用 `<KeepAlive :include="cachedViews" :max="5">` 包裹页面，由路由 `meta.keepAlive` 控制哪些页面进入 `cachedViews`。
- 页面需要写 `defineOptions({ name })`，`include` 按组件名匹配。
- 进入缓存的页面要逐个检查以下几点：
  - `onMounted` 里的首次加载保留；另加 `onActivated` 静默刷新，刷新期间不清空数据、不显示整表骨架。
  - 轮询、定时器、SSE、`window`/`document` 监听要在 `onDeactivated` 暂停，在 `onActivated` 恢复，否则页面在后台仍然会运行。
  - 已打开的弹窗、抽屉和临时表单，在切走时（`onDeactivated`）关闭或重置，防止回来时残留旧状态。
  - 与路由 query 同步的筛选条件（如运维页面）要确认激活时以当前 query 为准。
- 登出、切换账号或管理员合规状态变化时，清空全部缓存（例如给 `KeepAlive` 换 key）。
- 首批建议缓存：`AdminGroups`、`AdminAccounts`、`AdminUsers`、`AdminUsage`、`Keys`。`SettingsView` 表单状态复杂，暂不缓存。

验收：页面来回切换时不重复显示骨架；定时器数量不增长；登出后再登录看不到上一个身份的数据。

### 4. 内容区过渡与加载占位

目标：页面代码还没下载完时（预热未完成、网络慢），内容区显示骨架屏，而不是空白。

方案：

- 在 `AppShell` 内容区用 `<Suspense>` 或组件加载状态显示通用的页面骨架（标题栏 + 工具栏 + 表格占位），延迟约 150ms 再显示，避免快速切换时闪一下。
- 内容区加一个很短的淡入效果（≤150ms，遵守 `prefers-reduced-motion`）。这要和 glacier 主题现有的 `.workspace-content` 入场动画统一：外壳常驻后该动画只在首次挂载时播放，可以改为在内容区的子节点上播放。
- 列表页首次加载统一使用 `DataTable` 的 `loading` 骨架，不再显示空表格或 “暂无数据” 闪烁。

验收：慢 3G 下切换页面，全程看不到空白；开启减少动效后没有动画。

### 5. 拆分大页面

目标：减小首次打开大页面时的下载和解析量。

参考当前构建产物（gzip 前）：`AccountsView` 约 827 KB，`SettingsView` 约 415 KB，`OpsDashboard` 约 216 KB，`GroupsView` 约 214 KB。

方案：

- 把低频的大弹窗和编辑器改成 `defineAsyncComponent`，打开时再加载，例如：账号的创建/编辑/批量编辑/导入弹窗、分组编辑表单、设置页各个标签页的面板。
- `SettingsView` 按标签页拆成子组件，每个标签页异步加载。
- 用 `vite build` 的产物清单（可配合 `rollup-plugin-visualizer`）核对 chunk 体积，定一个上限，例如单个页面 chunk 不超过 300 KB。
- 拆分时要保持现有测试里对组件的 stub 名称和 `data-test` 选择器不变，避免大面积改测试。

验收：首次打开账号页和设置页的脚本体积明显下降；弹窗首次打开的延迟不超过 300ms（本地网络）。

## 建议顺序

先做 3，体感收益最大，但需要逐页检查生命周期；然后做 4，改动小、风险低；5 按页面体积从大到小逐步推进。
