# 渠道质量产物同步

固定映射为本地 Pro `72 ← mdkj 62`、Plus `237 ← mdkj 170`。只读取已完成的测试产物，不触发测试，不改变路由、计费、缓存或本站使用指标。`completed` 仅代表生成完成，供人工对比。

## 服务器安装

从已部署的精确提交复制本目录文件，运行：

```sh
install -d -m 700 /opt/sub2api-external-quality
install -m 700 sync.py /opt/sub2api-external-quality/sync.py
install -m 644 sub2api-external-quality.service sub2api-external-quality.timer /etc/systemd/system/
systemctl daemon-reload
```

将已授权 mdkj 会话的 `refresh_token` 通过 SSH 标准输入直接写到服务器 `/opt/sub2api-external-quality/mdkj.refresh`（0600），不要写入本地文件、命令参数、聊天或 Git。若源站要求验证码，由账号持有人在浏览器完成。同步程序自动轮换刷新令牌；撤销登录或源站拒绝刷新时需要重新授权，不尝试绕过验证码。

```sh
systemctl start sub2api-external-quality.service
systemctl enable --now sub2api-external-quality.timer
systemctl list-timers sub2api-external-quality.timer
cat /opt/sub2api-external-quality/status.json
```

定时器在上次执行结束后五分钟运行；单进程不重叠，超时180秒、内存上限96MiB。只允许固定 HTTPS 源站，拒绝重定向，JSON响应上限8MiB。

本站导入鉴权优先从服务器 `/opt/sub2api-external-quality/admin.key` 读取管理 Key 明文。独立管理 Key 的数据库记录仅有摘要，无法通过查库取回明文；在管理页创建或轮换后，须将明文经 SSH 标准输入直接写入该文件（root 所有、0600），不要写入本地文件、命令参数、聊天或 Git。当前受限权限目录未登记该导入路由，独立 Key 需具备现有 `full_access` 权限；受限 Key 会返回 403。Key 只由此脚本发往 localhost 导入端点。轮换或撤销后须同步更新文件。

仅在 `admin.key` 不存在时兼容从本地数据库读取旧版 `settings.admin_api_key`；空文件不会回退。两者均未配置时，状态文件和日志明确记录 `AdminIntegrationKeyMissing`，不再只是 `ValueError`。上游 RT 更新不能修复本站导入 Key 缺失。日志和状态文件只包含分组、数量、原始检测时间及错误类型，不包含密钥、令牌或 HTML；服务文件不携带凭据。

## 清理与展示

后端每次导入即去重、裁剪，每组每模型仅留最新10条，两个分组合计最多40条。现有V2 Aggregator启动及每8小时再次裁剪；生产不得禁用该 Aggregator。列表查询不加载HTML，用户展开产物才取单条，前端刷新时释放已过期产物缓存。数据库保留的是最近结果；同步故障时不清空最后结果，超过两小时明确标为待更新，不能将重新导入时间当作新检测时间。

渠道页面采用V1主动监控：仅两个分组，每5分钟小探测，显示真实可用率与完整响应延迟。缓存率单独来自本站近90分钟实际用量，排除带专用UA的新主动探测。完整响应延迟不标为首Token耗时。

产物在独立的opaque-origin iframe中运行内联动画和交互，仅启用 `sandbox="allow-scripts"`，不允许same-origin、顶层导航、弹窗、表单或下载。预览文档CSP限制外部资源、网络连接与嵌套框架，宿主CSP保持原样；暂停销毁预览框，重播重新创建。前台不输出内部来源、远端事件ID与账号信息；来源映射保留在服务端。DeepSeek仅移出外部质量结果，不改变手动缓存测试配置。

## 验证与停止

`python3 -m unittest discover -s deploy/external-quality -p 'test_*.py'` 使用隔离mock，不访问真实站点。

验收须检查两次真实同步、轮换凭据写入、最新检测时间、用户分组权限、产物按需加载及定时器启用状态；仅成功读取历史附件不算自动同步通过。

暂停抓取使用 `systemctl disable --now sub2api-external-quality.timer`，不会删除最后产物或共享数据库。旧应用镜像不等于数据库回滚；迁移337仅新增独立表。
