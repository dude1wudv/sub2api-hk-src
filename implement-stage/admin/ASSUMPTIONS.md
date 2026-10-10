# 管理端边界与决定

ASK mode: never（已确认方案，不重复授权）。

| ID | 决定 | 类别 | 来源 |
|---|---|---|---|
| A-001 | 真实业务字段全部保留；合成样稿只决定布局，不决定接口字段 | semantic | user |
| A-002 | 直接在当前工作区整合，不创建工作树，不清理既有修改 | interface | user |
| A-003 | 暖白管理端主题仅影响当前管理路由与其浮层，不覆盖保存的旧主题偏好 | interface | user |
| A-004 | 抽屉640px、编辑960px；布局断点1200/768px | interface | user |
| A-005 | 继续使用既有API client、Pinia和业务组件中的校验/请求，展示结构单独重排 | interface | user |
| A-006 | 所有现有平台及动态平台保留；不以样稿三个平台做白名单 | semantic | user |
| A-007 | 账号状态、额度与代理池汇总默认收起，完整内容可展开；主动进入额度池筛选时展开相关汇总 | interface | user |
| A-008 | 由用户端主任务统一提交部署，本侧通过 RELEASE_COORDINATION.md 交接 | interface | user |
| A-009 | 表单草稿只比较可编辑内容，远程能力开关、翻译提示和搜索词不计入；内容只留内存 | semantic | implementation |

技能审阅限制：本侧会话禁止子代理；独立假设审阅标记SWEEP_UNAVAILABLE，不能以主执行者自查冒充独立审阅。自动化检查与源码复核照常执行。
