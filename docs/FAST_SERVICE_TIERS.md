# OpenAI Fast 与 Ultra Fast

Fast 使用 `service_tier: "priority"`（客户端 `fast` 别名会被归一化）；Ultra Fast 使用 `service_tier: "ultrafast"`。两者适用于 OpenAI HTTP 请求及 WebSocket 每个 `response.create`。

OpenAI / Composite 分组可分别配置强制 Fast、强制 Ultra Fast。分组强制优先于全局 Fast/Flex 档位策略；同时开启时 Ultra Fast 优先。非档位的安全校验和模型准入仍生效。保存后新 HTTP 请求生效；已有 WebSocket 连接须重连以获取最新分组快照。

OpenAI Fast 默认按同一请求 Standard Token 价格的 2.5 倍计算，Ultra Fast 默认 8 倍。分组和渠道定价支持各自独立的 `fast_multiplier`、`ultrafast_multiplier`；未配置时使用默认值，显式配置必须为正数。命中的分组模型价卡整体优先于渠道，该卡档位倍率留空时回默认值，不继承渠道倍率；倍率不重复叠加。非 OpenAI 模型沿用原有 Fast 价格规则。

倍率只适用于 Token 成本；按次、图片、视频、搜索等独立计费项目沿用原价，不额外乘档位倍率。

分组的“免费 Fast”“免费 Ultra Fast”彼此独立，仅把对应档位的用户实际费用降至 Standard；上游请求仍使用对应加速档位。总成本及账号成本统计保留原有计算口径。上游降档计费沿用现有契约：公开 API 的明确低档响应可降低费用；Codex OAuth 的 `default` 响应沿用已有非权威判定，不把它直接作为降档依据。

网关保证转发字段，不保证账户、模型或上游实际提供某种速度。网关分别观察出站档位与响应档位，使用记录保存最终计费档位；降档可结合服务日志核对。

协议依据：[OpenAI Fast](https://developers.openai.com/api/docs/guides/fast-mode)、[OpenAI Ultrafast](https://developers.openai.com/api/docs/guides/ultrafast-mode)。2.5 倍和 8 倍为本站运营计费规则。
