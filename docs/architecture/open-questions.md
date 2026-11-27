# Open Questions

> 状态：Maintained
> 原则：状态为 `Accepted` 的 ADR 不得重新讨论；`Proposed` ADR 仍需负责人确认。
> 下列事项缺少产品或合规输入，不得自行编造。

## OQ-001：登录 Provider

- 问题：MVP 首先实现手机号、邮箱、第三方账号还是组合方式。
- 影响：身份 Provider、验证码供应商、注册流程和隐私字段。
- 推荐：MVP 实现邮箱或手机号中的一种主方式，并保留统一身份 Provider；
  第三方登录作为适配器后置。
- 阻塞：真实认证实现、验证码接口和注册合规文案。

## OQ-002：首发市场与内容分级

- 问题：首发地区、数据驻留、内容分级、年龄限制和实名要求。
- 影响：部署区域、上传审核、内容发布和隐私保留。
- 推荐：按中国大陆合规基线设计，同时保留区域配置。
- 阻塞：公开发布和后台审核规则。

## OQ-003：等待房聊天保留与审核

- 问题：聊天保留天数、是否需要全量机器审核、举报证据保存周期。
- 影响：`room_messages` 生命周期、审核队列和隐私政策。
- 推荐：MVP 保留较短周期，自动审核即时拦截，举报证据延长保存。
- 阻塞：数据库保留配置和公开 MVP 合规。

## OQ-004：运营角色粒度

- 问题：`content_reviewer`、`support_agent`、`security_auditor`、`platform_admin`
  是否足够，以及哪些操作需要双人复核。
- 影响：Admin API、权限矩阵、审批 UI 和审计。
- 推荐：MVP 使用四角色 RBAC 加 scope ABAC；D4 访问和封禁支持双人复核。
- 阻塞：后台前后端实现。

## OQ-005：AI 进度事件持久化

- 问题：`ai_progress` 的 `seq` 在 Worker 重启和断线后需要保留多久。
- 影响：进度表、补偿策略、SSE 重连和清理。
- 推荐：在 `generation_jobs.progress_seq` 和步骤投影中持久化最新状态；
  完整中间进度不作为长期审计数据。
- 阻塞：生成进度契约和测试 fixture。

## OQ-006：邀请 token 规则

- 问题：邀请最大使用次数、过期时长、是否限定账号、是否允许转发。
- 影响：`invitation_tokens`、API R07/R08 和并发兑换。
- 推荐：MVP 支持有限次数、可撤销、短有效期；默认不限定账号。
- 阻塞：房间邀请实现和数据库迁移。

## OQ-007：首页聚合、通知、成就与完整历史

- 问题：这些页面的 MVP 完整度和数据来源。
- 影响：API、数据库和 UI。
- 推荐：首页使用公开搜索兜底；近期历史先实现；通知、成就和完整历史不阻塞核心对局。
- 阻塞：完整页面验收，不阻塞剧本杀主闭环。

## OQ-008：AI 点评展示承诺

- 问题：公开 MVP 是否要求 AI 点评可用，还是只要求结构化复盘。
- 影响：P1 异步任务、结算页和成本指标。
- 推荐：规则结果与结构化复盘为 P0；AI 点评异步增强。
- 阻塞：结算页最终验收文案。

## OQ-009：图片生成配额与供应商

- 问题：默认是否关闭、每用户配额、备用 Provider 和内容审核责任。
- 影响：成本、安全、任务超时和素材生产。
- 推荐：MVP 默认关闭；开启时按作品和用户双重配额。
- 阻塞：图片 Provider 合同和生产配置。

## OQ-010：数据保留与删除

- 问题：私聊、完整 AI 上下文、事件、审计和账号注销的精确保留天数。
- 影响：生命周期 Worker、合规和复盘。
- 推荐：采用 DATABASE_DESIGN 的区间值作为默认，最终由产品、安全和法务签署。
- 阻塞：公开发布的数据合规评审。

## OQ-011：Agent 任务级输出 Schema

- 问题：AI DM、NPC、规则解释和剧本生成分别允许哪些输出字段、引用和动作候选。
- 影响：Python Schema Registry、Go 二次校验、事件投影和前端展示。
- 推荐：每个任务独立 Pydantic/JSON Schema，统一禁止任意状态补丁；Go 保留最终权限和规则判断。
- 阻塞：除 Demo 外的 Agent 业务接入。

## OQ-012：内部 AI 传输认证

- 问题：生产环境是否强制 mTLS、Service Token、body hash、timestamp 和 nonce。
- 影响：Go InternalAIClient、Python 中间件、证书轮换和可观测性。
- 推荐：内测使用短 token，生产强制 mTLS 加请求签名。
- 阻塞：跨信任域或公网附近部署。

## OQ-013：MCP 工具信任与审计

- 问题：哪些 MCP Server 可信、工具如何发现、权限如何映射、结果如何审计和限流。
- 影响：Tool Registry、安全边界、网络访问和运营审计。
- 推荐：只允许显式白名单 Server 与工具，禁止模型任意发现或执行。
- 阻塞：真实 MCP 接入。

## OQ-014：AI Run 持久化与 Checkpoint 保留

- 问题：`mv_ai` Run/Step/Tool/Usage 与 LangGraph checkpoint 的保留周期和恢复目标。
- 影响：故障恢复、成本归因、审计和存储。
- 推荐：Run 摘要长期保留，完整上下文短期加密保存，Checkpoint 按任务生命周期清理。
- 当前：DQ-21 已批准；GitHub Actions Run `45803587621` 已通过真实 MySQL/Redis
  Migration、Persistence、Multi-Worker、Recovery/Cleanup 和 Backup/Restore。
- 剩余：Production、真实 KMS、RTO/RPO、容量和 Graph Auto Replay 仍
  `NOT VERIFIED`。

## OQ-015：内部请求签名最终规范（已解决）

- 问题：是否生产强制 mTLS + HMAC，签名 canonical 字段、nonce 存储、时钟偏差和密钥轮换周期如何冻结。
- 影响：Go Client、Python 中间件、multitenancy nonce 存储、证书和 KMS。
- 结论：DQ-19 已批准。生产目标为 mTLS + `mv-hmac-sha256-v1`；canonical 绑定
  signature version、Key ID、timestamp、nonce、Service ID、method、path、query
  和 body hash；生产 nonce 使用 Redis 原子消费。
- 剩余：真实 CA、KMS、Redis 多实例和生产证书轮换仍为 `NOT VERIFIED`。

