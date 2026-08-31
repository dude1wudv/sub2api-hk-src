# Sub2API 前端视觉优化与实施计划

> 状态：提案 v1（仅规划，尚未实施）
> 目标：在现有 `warm-paper` 自定义主题基础上，建立更精致、克制、可持续维护的开发者控制台视觉系统；不做 Vercel/Linear 的简单仿制，也不改变业务行为。

## 1. 项目目标与边界

### 1.1 目标

1. 保留当前“暖纸、陶土主色、冷蓝链接”的品牌辨识度，提升细节、层级和一致性。
2. 将当前零散的颜色替换升级为可维护的语义化设计 Token。
3. 优先打磨高频界面：应用框架、登录、用户仪表盘、API Key、管理仪表盘与数据表格。
4. 同时完善浅色、暗色、响应式、键盘操作、焦点状态、空/错/加载状态。
5. 通过分批交付、视觉快照和现有自动化测试控制回归风险。

### 1.2 非目标

- 不改变 API、Store、路由、权限、计费或表格数据逻辑。
- 不在第一轮重写组件库或升级 Vue/Tailwind。
- 不为了“高级感”大面积引入玻璃拟态、模糊、动态粒子或重动画。
- 不将暖色品牌主题整体替换成通用黑白/冷灰模板。
- 不一次性重做全部 204 个 Vue 组件和 86 个页面。

## 2. 当前基线审计

### 2.1 已有优势

- `frontend/src/styles/themes/warm-paper.css` 已形成清晰的品牌起点：
  - 陶土橙 `primary`
  - 冷蓝 `link`
  - 独立的成功、警告、错误色阶
  - `canvas / surface / ink / line` 基础语义变量
- `frontend/tailwind.config.js` 已支持 CSS Variable 色阶，可渐进迁移而无需推翻 Tailwind。
- `frontend/src/style.css` 已有按钮、输入框、卡片、表格、弹窗、Toast、侧栏等基础组件类。
- `AppSidebar.vue`、`AppHeader.vue`、`TablePageLayout.vue` 已具备桌面/移动、折叠、固定表头等成熟行为。
- 已有暗色模式、`prefers-reduced-motion` 局部支持以及较完整的 Vitest 测试基础。

### 2.2 主要问题

1. **主题只覆盖了一部分视觉层**
   `warm-paper.css` 主要覆盖基础面、主按钮和侧栏，业务组件仍大量直接使用 `gray-*`、`dark-*`、`blue-*` 等类，最终效果不完全统一。

2. **物理色与语义色混用**
   `tailwind.config.js` 将 `blue/sky/cyan/teal/indigo` 都映射到 `link`，将 `purple/violet/pink` 映射到 `primary`，将多种绿映射到 `ok`。这对状态语义有利，但会让平台徽章、图表序列和数据分类失去必要的色相区分。

3. **装饰语言略显过量**
   主按钮、登录页和首页同时使用大圆角、渐变、光晕、玻璃、模糊球和网格背景，容易形成早期 SaaS/Web3 风格，削弱“稳定 API 基础设施”的专业感。

4. **密集数据页层级不够稳定**
   管理仪表盘使用多色指标图标；表格、过滤器、卡片、图表的圆角、边界和间距不完全一致。高密度场景中颜色承担了过多层级职责。

5. **视觉字面量较多**
   粗略扫描 `frontend/src` 中 `.vue/.css` 文件可见约 581 处 Hex/RGB 字面量；大量组件直接引用具体色阶。不能一次性替换，但需要建立迁移规则。

6. **暗色仍偏棕且层级较近**
   当前暗色主题具有特色，但 `canvas/surface/surface-2/line` 的明度距离较小，部分复杂表格、浮层和输入框可能显得“糊在一起”。应提升层级而不是简单改成纯黑。

## 3. 对标学习结论

### 3.1 Linear：降低噪音、提高导航密度

可迁移细节：

- 侧栏、标签、页头和面板遵循同一对齐基线。
- 通过减少不必要图标/色彩、弱化侧栏、提高内容对比来建立层级。
- 先用主视图做压力测试，再定义组件行为，随后灰度发布，而不是一次全站替换。
- 使用变量生成 surface、text、icon、control 别名，主题差异由 Token 承担。

不照搬：不采用其品牌紫色和纯冷黑背景。

### 3.2 Stripe：颜色可访问且保持品牌感

可迁移细节：

- 色阶不仅“看起来协调”，还应按前景/背景组合预先验证对比度。
- 使用感知更均匀的颜色模型辅助校准不同色相的视觉重量。
- Badge、状态色和图表色要同时满足可区分性与文字对比度。

不照搬：不引入 Stripe 品牌紫蓝渐变。

### 3.3 Vercel Web Interface Guidelines：交互细节与可访问性

可迁移细节：

- 所有流程可键盘操作，使用 `:focus-visible` 提供明确焦点。
- 移动端点击目标建议达到 44px；桌面端至少满足 WCAG 2.2 的 24px 目标要求或安全间距例外。
- 数字比较使用 `tabular-nums`；API Key、模型名、耗时等适度使用等宽字体。
- 状态不能只靠颜色表达，应同时有文字、图标或形状。
- 动画只用于解释因果，显式声明动画属性，避免 `transition: all`，并支持 `prefers-reduced-motion`。

### 3.4 OpenAI Platform / Anthropic Console：信息架构启发

可迁移细节：

- 把 API Key、Usage、Billing/Balance、模型/渠道状态作为用户端核心任务。
- 默认视图突出“现在是否可用、花了多少、如何开始”，详细运营数据按需展开。
- 复制 Key、错误状态、限额和用量反馈应就地呈现，不依赖用户猜测。

不照搬：本轮不改变导航信息架构和业务流程，只优化现有路径的视觉与反馈。

## 4. 推荐视觉方向：Warm Precision

关键词：**暖纸、精密、克制、可信、数据优先**。

### 4.1 视觉原则

1. **暖色是品牌，不是装饰**：陶土色只用于主行动、选中态和少量品牌强调。
2. **冷蓝是交互**：链接、可点击数据和信息提示统一使用 `link`，避免与主行动竞争。
3. **状态色只表达状态**：绿色=成功、琥珀=警告、红色=错误；普通分类不借用状态色。
4. **层级优先靠明度、边界、间距**：阴影只用于浮层和必要的悬浮关系。
5. **数据优先**：数字使用 `tabular-nums`，必要时使用等宽字体；图表颜色克制但可区分。
6. **浅暗色独立校准**：暗色不是简单反转浅色，保留暖调但提高表面分层和文字对比。

### 4.2 Token 结构

在保留现有色阶的同时，新增/规范以下语义别名：

```css
:root {
  --bg-app: var(--canvas);
  --bg-surface: var(--surface);
  --bg-subtle: var(--surface-2);
  --bg-elevated: 255 254 251;

  --text-primary: var(--ink);
  --text-secondary: var(--ink-2);
  --text-muted: var(--ink-3);

  --border-subtle: var(--line);
  --border-default: var(--line-strong);
  --focus-ring: var(--color-link-500);

  --radius-control: 0.5rem;
  --radius-card: 0.75rem;
  --radius-overlay: 0.875rem;
}
```

Token 分层：

1. **Primitive**：陶土、蓝、绿、琥珀、红、暖中性色阶。
2. **Semantic**：背景、表面、文字、边界、交互、状态、焦点。
3. **Component**：按钮、输入框、卡片、侧栏、表格、浮层。

关键决策：

- `primary/link/ok/warn/err` 继续作为语义色。
- 恢复一套独立的数据可视化/平台分类色，避免所有蓝紫绿被折叠成 4 个语义色。
- 业务组件逐批迁移，不进行高风险的全局字符串替换。
- 候选 Token 必须经过真实页面和对比度检测后定稿，计划中的色值不是未经验证的最终规范。

### 4.3 几何、阴影、排版与动效

| 项目 | 现状倾向 | 目标 |
| --- | --- | --- |
| 控件圆角 | 12–16px 较多 | 8px 为主 |
| 卡片圆角 | 16px 较多 | 10–12px |
| 浮层圆角 | 16px | 12–14px |
| 卡片阴影 | 常态可见 | 默认边框，hover 极轻阴影 |
| 主按钮 | 渐变 + 彩色光晕 | 低幅渐变或纯色，弱阴影 |
| 页面背景 | 多个模糊球/网格 | 保留一个低对比品牌纹理 |
| 标题 | 粗体偏多 | 600 为主，明确字号层级 |
| 数据 | 普通比例数字 | `tabular-nums`，关键数据可用 mono |
| 动画 | `transition-all` 较多 | 只过渡 color/opacity/transform/shadow |

## 5. 组件与页面实施方案

### Phase 0：基线与视觉实验台

**目的**：先建立可比较、可回滚的基线。

交付物：

- 记录 360×800、768×1024、1440×900、1920×1080 四档视口。
- 记录浅/暗色的登录、用户仪表盘、Keys、Usage、管理仪表盘、Accounts、Settings 截图。
- 建立一个仅开发环境使用的视觉样板页，覆盖按钮、输入、Select、Badge、Toast、Modal、Card、Table、状态、空/错/加载状态。
- 记录当前 `pnpm typecheck / lint:check / critical Vitest / build` 基线。

验收：没有业务代码改动；能稳定复现改造前视觉。

### Phase 1：Token 与基础组件

**主要路径**：

- `frontend/src/styles/themes/warm-paper.css`
- `frontend/src/style.css`
- `frontend/tailwind.config.js`
- `frontend/src/components/common/**` 中实际复用的基础控件

工作项：

1. 完善 surface/text/border/focus/component Token。
2. 分离“状态色”和“数据/平台分类色”。
3. 收敛按钮、输入、卡片、Badge、Tabs、Dropdown、Modal、Toast 的圆角、边界、阴影和状态。
4. 补全 hover/active/focus-visible/disabled/loading/error。
5. 全局减少无意的 `transition-all`，补充 reduced-motion。

验收：视觉样板页浅/暗色一致；正常文本达到 WCAG AA 4.5:1，大文本至少 3:1；不依赖颜色单独表达状态。

### Phase 2：应用框架与认证

**主要路径**：

- `frontend/src/components/layout/AppLayout.vue`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/components/layout/AppHeader.vue`
- `frontend/src/components/layout/AuthLayout.vue`
- `frontend/src/views/auth/LoginView.vue`
- `frontend/src/views/auth/RegisterView.vue`
- `frontend/src/views/auth/ForgotPasswordView.vue`
- `frontend/src/views/auth/ResetPasswordView.vue`

工作项：

1. 统一侧栏、页头、页面内容的水平基线和密度。
2. 保留展开/折叠和移动抽屉行为，仅优化选中态、分组层级、图标与文字对齐。
3. 精简 Header 控件视觉，确保余额、订阅、通知、语言和用户菜单在小屏不拥挤。
4. 认证页减少光晕与背景元素，突出品牌、标题、表单和主操作。
5. 表单错误、密码显示、OAuth、Turnstile 状态保持清晰。

验收：保留所有路由、DOM Hook、ARIA 标签和 Onboarding ID；键盘可完成登录；移动端输入字号和点击目标符合要求。

### Phase 3：用户端高频路径

**主要路径**：

- `frontend/src/views/user/DashboardView.vue`
- `frontend/src/components/user/dashboard/**`
- `frontend/src/views/user/KeysView.vue`
- `frontend/src/views/user/UsageView.vue`
- `frontend/src/views/user/ProfileView.vue`
- `frontend/src/views/user/PaymentView.vue`
- `frontend/src/views/user/SubscriptionsView.vue`

工作项：

1. 仪表盘指标去“彩虹化”，按主指标、趋势、辅助信息建立层级。
2. 图表统一暖品牌 + 冷交互 + 可区分数据色；补充无数据和异常说明。
3. API Key 列表优先优化创建、复制、掩码、状态、额度与到期信息。
4. 统一充值/订阅/支付页的金额层级、选中态和风险提示。
5. 数字、金额、Token、延迟统一使用 `tabular-nums`。

验收：核心操作步数不增加；长 Key、长模型名、中文/英文、小屏均不溢出；复制和状态反馈对键盘及读屏可感知。

### Phase 4：管理后台与数据密集页

**主要路径**：

- `frontend/src/views/admin/DashboardView.vue`
- `frontend/src/views/admin/AccountsView.vue`
- `frontend/src/views/admin/GroupsView.vue`
- `frontend/src/views/admin/UsersView.vue`
- `frontend/src/views/admin/UsageView.vue`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/components/layout/TablePageLayout.vue`
- 对应 `frontend/src/components/account/**` 与通用表格组件

工作项：

1. 统一过滤器、批量操作、表格、分页和详情弹窗的密度。
2. 管理仪表盘按“系统健康 → 今日流量/成本 → 渠道余额 → 趋势”重排视觉层级，但不改变数据请求。
3. 表格行高目标 40–44px；复杂单元格允许 48px，不盲目压缩。
4. Sticky header/column、横向滚动、移动卡片模式保持现有行为。
5. 平台状态、额度、错误和临时停调使用图标/文字/形状冗余表达。

验收：现有筛选、排序、批量操作、Sticky 和滚动行为不变；100%/125%/200% 缩放可用。

### Phase 5：首页与收尾

**主要路径**：

- `frontend/src/views/HomeView.vue`
- `frontend/src/views/ModelPlazaView.vue`
- 空状态、404、引导 Tour、公共法律页

工作项：

1. 将首页装饰收敛为一套品牌背景，避免多层模糊球、网格和玻璃同时竞争。
2. 统一 Provider 卡片、终端演示和 CTA 风格。
3. 清理迁移后废弃样式和重复 Token。
4. 完成视觉回归矩阵、可访问性检查和最终性能检查。

## 6. 父代理与 Gemini 3.7 Flash 协作方式

### 6.1 角色

**父代理（总协调）**：

- 冻结范围、拆分批次、保护业务边界。
- 维护验收清单，审查 Gemini 的改动与残余风险。
- 运行测试、处理跨组件冲突、决定是否进入下一阶段。
- Git 提交、推送、部署与生产操作始终由父代理负责，并在需要时取得用户确认。

**Gemini 3.7 Flash（前端制作）**：

- 按单一 Phase/单一文件边界实施，不跨批次自由重构。
- 每次先给出拟改文件、视觉目的、保留行为，再编码。
- 不修改 API、Store、路由、业务数据结构和生产配置。
- 每批返回：改动文件、组件状态覆盖、测试结果、截图/人工检查点、剩余问题。

### 6.2 每批工作循环

1. 父代理提供目标、允许文件、禁止项、参考截图和验收条件。
2. Gemini 输出微型设计说明与实施清单。
3. Gemini 完成该批代码和局部测试。
4. 父代理检查 diff、类型、Lint、测试、浅/暗色和响应式。
5. 不达标则在同一子代理会话定向修正；达标后才进入下一批。

原则：同一工作区同时只保留一个写入代理，避免覆盖当前大量未提交改动。

## 7. 质量门禁

### 7.1 自动化

每批至少执行：

```bash
pnpm --dir frontend run typecheck
pnpm --dir frontend run lint:check
pnpm --dir frontend exec vitest run <本批相关 spec>
```

阶段结束执行：

```bash
pnpm --dir frontend run build
make test-frontend-critical
```

最终阶段再评估是否运行全量 Vitest。

### 7.2 视觉矩阵

| 维度 | 覆盖 |
| --- | --- |
| 主题 | Light / Dark |
| 视口 | 360×800 / 768×1024 / 1440×900 / 1920×1080 |
| 缩放 | 100% / 125% / 200% |
| 语言 | 简体中文 / English |
| 状态 | 默认 / hover / focus / active / disabled / loading / empty / error |
| 输入 | 鼠标 / 键盘 / 触控 |

### 7.3 可访问性验收

- 正常文本对比度 ≥ 4.5:1；大文本 ≥ 3:1。
- 重要非文本 UI 边界/状态达到可辨识对比。
- 桌面焦点目标不小于 24×24px 或满足 WCAG 间距例外；移动端目标优先达到 44×44px。
- 焦点可见且不被 Sticky Header、弹窗或抽屉遮挡。
- 状态不只依赖颜色；图表提供文字摘要或等价数据。
- `prefers-reduced-motion` 下取消非必要位移、缩放、呼吸和自动动画。

## 8. 风险与控制

| 风险 | 控制措施 |
| --- | --- |
| 当前工作区存在大量未提交改动 | 每批限定文件；改前检查 `git diff`；不回滚用户改动 |
| 全局 Token 导致远端页面意外变色 | 先视觉样板，再核心页面，最后全站；禁止一次性颜色替换 |
| 状态色映射破坏平台区分 | 独立保留 semantic status 与 categorical palette |
| 暗色主题对比不足 | 真实组件组合检测，不只检查单个色值 |
| 高密度表格改造破坏 Sticky/横向滚动 | 保留 DOM 与布局行为，先只改视觉层 |
| Gemini 越界重构业务 | 每批明确 allowlist/denylist，由父代理审查 diff |
| 上游 Sub2API 合并冲突 | 自定义视觉尽量集中在主题和基础组件，减少散落覆写 |

## 9. 建议里程碑

| 里程碑 | 内容 | 预估工作量 |
| --- | --- | --- |
| M0 | 基线截图、视觉样板、Token 定稿 | 0.5–1 天 |
| M1 | 基础组件、App Shell、认证页 | 1.5–2.5 天 |
| M2 | 用户仪表盘、Keys、Usage | 2–3 天 |
| M3 | 管理仪表盘、Accounts、表格体系 | 3–5 天 |
| M4 | Settings/支付/订阅/首页及收尾 | 2–4 天 |

总计约 9–15 个有效工作日；应按里程碑验收，不建议一次性连续改完后再检查。

## 10. 第一批建议任务

实施从 M0 开始，不直接改首页：

1. 建立视觉样板和基线截图清单。
2. 定稿 `warm-paper` 的浅/暗色 Token 与分类色策略。
3. 先打磨 `Button / Input / Card / Badge / Table / Modal`。
4. 应用到 `AppSidebar + AppHeader + LoginView` 做首个端到端样板。
5. 用户确认整体方向后，再进入用户仪表盘与 Keys 页。

## 11. 参考资料

- Linear, *How we redesigned the Linear UI (part II)*
  https://linear.app/now/how-we-redesigned-the-linear-ui
- Vercel, *Web Interface Guidelines*
  https://vercel.com/design/guidelines
- Stripe, *Designing accessible color systems*
  https://stripe.com/blog/accessible-color-systems
- W3C, *How to Meet WCAG 2.2 (Quick Reference)*
  https://www.w3.org/WAI/WCAG22/quickref/
- Anthropic Console
  https://platform.claude.com/dashboard
- OpenAI developer platform
  https://platform.openai.com/

---

本计划的核心判断：**Sub2API 不需要变成另一个黑白开发者工具；更好的方向是把现有暖纸主题从“换色皮肤”升级为完整、克制、可验证的 Warm Precision 设计系统。**
