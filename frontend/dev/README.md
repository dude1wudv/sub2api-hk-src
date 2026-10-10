# patrickapi 本地合成预览

在 `frontend` 目录执行：

```powershell
node dev/user-preview.mjs
```

需要已安装项目的 Node.js 与 pnpm 依赖。脚本不安装包、不读取 `.env`、不读取真实后端配置、不连接生产服务。浏览器打开 `http://127.0.0.1:3410/home`；mock API 仅监听 `127.0.0.1:3411`。端口占用即停止，不换端口、不寻找真实后端。`Ctrl+C` 同时关闭两项服务。

| 用途 | 合成账号 | 演示密码 |
| --- | --- | --- |
| 新版用户工作区 | `preview@example.test` | `PreviewOnly123!` |
| 现有管理外壳 | `admin@example.test` | `PreviewOnly123!` |

这些账号、密码、token 和 `sk-preview-` 密钥都是本地演示值，对任何真实服务均无效。请勿在预览中输入真实凭据。前端继续通过原认证流程将假 token 存入此本机 origin 的浏览器存储；服务端会话和修改仅在内存中，重启后需要重新登录，密钥修改也会恢复初始数据。脚本不记录请求正文、密码、密钥或授权头，不将会话/修改保存到文件。

## 已覆盖的界面

- 首页、登录、注册/找回密码页面的展示和公开设置；登录、刷新与退出可操作。
- 用户 `/dashboard`：兼容进入新版 API 页面。
- `/keys`：分页、搜索/状态/分组筛选、创建、编辑、启停、删除；仅接受合成 `sk-preview-` 自定义密钥。
- `/usage`：合成日志、分页、主要筛选、统计、图表、详情、密钥每日用量。
- `/profile`：账户、通知设置、绑定状态和安全功能读取；更改账户、密码或绑定不执行。
- `/model-plaza`：三类分组和合成模型价格；`embedded=1` 仍遵守原登录状态判断。
- `/key-usage`：从本预览的密钥页复制 `sk-preview-` 密钥，可查询合成用量。
- `/purchase`、`/redeem`：购码外链展示与合成兑换。使用 `PREVIEW-TOPUP-10` 或 `PREVIEW-TOPUP-50` 增加本地余额，每个合成用户每种码仅可使用一次；重启恢复。
- `/orders`：本站合成兑换历史。`/subscriptions`：合成订阅和套餐，没有真实支付方式。
- `/monitor`：合成 v2 监控概览、维度、模型与矩阵；错误/用户列表为空。
- 管理员 `/admin/dashboard`：用于验收旧外壳的基本合成统计；管理设置只提供导航所需的公开字段，不模拟完整运维系统。

未实现的接口明确返回 `501 PREVIEW_UNAVAILABLE`，不会伪装成成功。注册、密码重置、OTP、OAuth、Passkey、创建订单、订阅购买、退款、真实兑换、外部跳转及其他真实写操作均不可用。密钥 CRUD 与上述两种合成兑换码是业务写入模拟，限于各合成用户自己的内存记录。

## 隔离方式与边界

- Vite 使用 `configFile: false`、`envFile: false`，显式载入 Vue 插件、`@` 别名、运行版 `vue-i18n` 和 JIT 标志；保留项目的正常 Tailwind/PostCSS 样式处理。
- `VITE_API_BASE_URL` 固定为相对路径 `/api/v1`，`/api`、`/v1`、`/setup`、`/health` 代理固定到本机 mock。公开设置中的 `https://patrickapi.microedulab.com` 仅用于产品接入地址展示。
- HTML 注入 `window.__APP_CONFIG__` 和“本地预览 · 合成数据”标识，生产源码没有此标识。另加只允许同源资源的 CSP、外链/表单/`window.open` 拦截；示例中的外部调用不会执行。
- 两服务都只绑定 `127.0.0.1` 并检查本机 Host；mock 不包含任何转发、上游请求或数据库连接代码。
- 这套预览用于 UI 和交互验收，不验证真实认证安全、模型推理、支付、服务器监控或正式接口完整性。模型名、余额、价格、请求和健康状态均为合成示例。

静态语法检查（不会启动服务）：

```powershell
node --check dev/user-preview.mjs
node --check dev/user-preview-fixtures.mjs
```
