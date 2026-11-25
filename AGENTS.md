# MysteryVerse 项目开发规范

> 适用范围：本文件位于仓库根目录，是 Codex 与所有开发者在本仓库工作时的默认规范。
>
> 基线依据：`docs/MysteryVerse-PRD.md`、`docs/TECH_DESIGN.md`、`docs/DATABASE_DESIGN.md`、`docs/API_DESIGN.md`、`docs/UI_DESIGN.md`、`docs/TASKS.md`、`docs/DESIGN_REVIEW.md`。
>
> 架构基线：`docs/decisions/README.md`、`docs/architecture/`、`docs/api/`、
> `docs/database/`。
>
> 决策状态与批准门禁：`docs/architecture/DECISION-BASELINE.md`。
>
> 文档冲突时：先停止实现并登记冲突，不得为了实现方便静默选择其中一种说法。

## 1. 项目定位

MysteryVerse（谜境）是一个 AI Native 多人互动推理游戏平台，包含：

- AI 剧本杀。
- AI 狼人杀。
- 沉浸式互动 RPG。

首个 MVP 优先实现 AI 剧本杀完整闭环：

```text
注册登录
→ 发现或导入剧本
→ AI 解析、结构化与一致性校验
→ 人工修正和发布
→ 创建并邀请房间
→ 角色分配与权限投影
→ 搜证、讨论、质询、投票
→ 确定性结算
→ 真相复盘
→ 可监控、可审核、可回滚
```

MVP 不包含狼人杀、RPG、AI 真人补位、语音、持久共享世界、付费交易和原生 App。

## 2. 文档路径基线

| 文档 | 仓库实际路径 |
| --- | --- |
| PRD | `docs/MysteryVerse-PRD.md` |
| 技术设计 | `docs/TECH_DESIGN.md` |
| 数据库设计 | `docs/DATABASE_DESIGN.md` |
| API 设计 | `docs/API_DESIGN.md` |
| UI 设计 | `docs/UI_DESIGN.md` |
| 任务清单 | `docs/TASKS.md` |
| 设计审查 | `docs/DESIGN_REVIEW.md` |

`docs/PRD.md` 当前不存在。不得继续在代码、脚本或任务说明中引用该错误路径。

外部 API 资源名统一使用数据库已收敛的 `script` / `script_version`。`Project` 仅可作为产品语境同义词，不得进入新 DTO、数据库字段、协议字段或生成类型，除非完成正式架构决策。

## 3. 技术架构

具体版本和细节以 `docs/TECH_DESIGN.md` 为准。已明确的技术边界如下。

| 层 | 技术与职责 |
| --- | --- |
| 前端 | React + TypeScript + Vite + Tailwind CSS；Zustand 保存 UI 与实时投影副本；TanStack Query 管理 REST 服务端状态；Motion 负责视觉动效；React Router 负责路由体验 |
| Go 核心业务 | Chi + `net/http`；`sqlc` + `database/sql`；`goose` 迁移；模块化单体部署为 API、Realtime/Runtime、Worker 三个进程 |
| AI 编排 | Python 3.12+、FastAPI、LangGraph、Pydantic v2、`httpx`、`redis.asyncio`、SQLAlchemy 2 Async Core、OpenTelemetry |
| 数据库 | MySQL 8.0.30+；权威业务事实、状态版本、事件、结算和审计 |
| 缓存与实时协调 | Redis 7.0+；Session 缓存、Presence、租约、Pub/Sub、Streams、热投影和短期记忆 |
| 对象存储 | S3 兼容存储；开发环境使用 MinIO；保存原始文件、生成产物、大快照和复盘导出 |
| 公共契约 | `packages/contracts`；OpenAPI、JSON Schema、WS/SSE、GamePackage、AI Output 的唯一契约源 |

约束：

- `packages/contracts` 先于 Go、TS、Python 实现。生成代码不得手改。
- 对外 JSON 使用 `snake_case`；API 根路径为 `/api/v1`；时间为 UTC RFC 3339。
- WebSocket 使用 `wss://.../ws/v1` 与 `mvws.v1` 子协议。
- 浏览器只访问 Go 公共 API、WebSocket 和 Go 代理的 AI 流。Python AI 服务不暴露公网。
- MVP 采用模块化单体，不为用户、内容、房间、游戏等提前拆成网络微服务。
- 未经技术评审不得新增框架、数据库、消息系统、向量数据库、gRPC 全量协议或第二套 AI 编排。

## 4. 服务职责与权威边界

### 4.1 Go：权威游戏业务状态

Go 是用户、内容状态边界和权威游戏状态的唯一业务写入者，负责：

- 用户与权限。
- 房间管理。
- 玩家管理。
- 游戏生命周期。
- 规则引擎。
- 角色分配。
- 线索权限。
- 投票。
- 胜负判定。
- WebSocket。
- 游戏事件与持久化。
- 服务端 DTO/Viewer/Private 投影。
- 权威上下文裁剪与 AI 调用编排。

### 4.2 Python：AI 能力

Python 负责：

- 剧本生成。
- 剧本结构化。
- LangGraph 工作流。
- 内容一致性检查。
- 自动修复。
- AI DM。
- NPC 对话。
- AI Memory。
- AI 玩家行为候选。
- 复盘文本与引用。
- Prompt、模型路由、Usage、Trace 和评测。

Python 只能读取 Go 授权的知识切片，并只能提交结构化文本、动作候选、关系候选、Memory Candidate 或 Artifact 引用。

### 4.3 Python 强制禁止

Python 不得：

- 直接修改 Go 的权威游戏状态。
- 直接写房间、游戏、玩家、角色、投票、技能、事件或结算表。
- 直接发布剧本版本或广播正式系统事件。
- 直接访问 `mv_core.secret_payloads` 或绕过 SecretService。
- 读取未由 Go 裁剪的完整知识库、身份表、私密事件或 NPC 记忆。
- 生成并执行任意 SQL、Go/Python 代码或未注册规则。

### 4.4 数据权威顺序

1. MySQL 权威事务、当前状态、事件日志、结算和审计。
2. 对象存储中的不可变内容包、快照和生成产物。
3. Redis、内存、客户端投影等可丢失、可重建数据。

Redis 不是权威业务数据源。Redis 丢失后必须能从 MySQL、对象存储和事件日志恢复。

## 5. 架构红线

以下规则不可突破：

1. 同一业务事实只有一个权威写入者。
2. 同一 `game_id` 的状态变更必须线性化；版本和事件序号由服务端分配。
3. 客户端只能提交意图和 `expected_version`，不得提交完整新状态、自造事件或胜负。
4. 身份分配、线索权限、投票、技能、胜负和揭示门控必须由 Go 确定性代码执行。
5. LLM 不得直接修改权威游戏状态，不得参与确定性规则执行。
6. 所有客户端可见数据必须由服务端完成对象 ACL、字段 ACL、reveal gate 和 Viewer 投影。
7. 前端隐藏字段、路由守卫、频道名、Query Key 和按钮禁用都不是权限。
8. 所有游戏意图必须幂等；作用域至少包含 `game_id + idempotency_key`。
9. 事务提交前不得广播成功事件；失败校验不得修改状态、消耗资源或伪造成功。
10. 游戏事件不可原地修改；错误通过新补偿事件修正。
11. AI 生成失败不得覆盖上一可用版本；发布必须经过 staging、校验和原子切换。
12. LangGraph 只负责内容生成工作流，不负责在线游戏规则裁决。
13. 本地磁盘不得作为跨实例权威存储。
14. D2-D4 正文不得进入普通日志、公共缓存、公开对象或未授权 DTO。
15. 涉及巨大文件的删除前，必须先向用户说明范围、影响与替代方案，并取得明确确认。

## 6. 公共契约与协议

### 6.1 契约源

```text
packages/contracts/
├── openapi/
├── json-schema/
│   ├── game-package/
│   ├── game-events/
│   ├── ws-messages/
│   └── ai-outputs/
├── examples/
├── compatibility/
└── codegen.yaml
```

- 当前唯一权威源为 `packages/contracts/json-schema/mysteryverse.schema.json`。
- 生成命令为 `scripts/contracts.ps1 generate`，漂移与三端测试为
  `scripts/contracts.ps1 check`。
- 禁止绕过 `packages/contracts` 随意定义跨服务数据结构。
- 禁止手改生成目录中的 Go、TS、Python 类型。
- Schema 使用 SemVer；不兼容变更必须新增 Major 或显式迁移路径。
- 所有 JSON 扩展字段必须带 `schema_version`，并在写入前校验。
- OpenAPI operation 必须有唯一 `operationId`、明确鉴权和统一错误响应。
- 请求 schema 默认 `additionalProperties: false`；未知字段策略必须明确。

### 6.2 API 与实时协议

- 成功使用 `data + meta`；失败使用稳定 `error.code`、安全 `details`、`retryable` 和 `request_id`。
- 所有非 GET 持久写请求必须支持 `Idempotency-Key`。
- 游戏写请求必须包含 `expected_version`。
- 同一幂等键不同 payload 返回 `IDEMPOTENCY_CONFLICT`。
- AI 超时、Schema 失败、安全阻断、Provider 不可用必须有结构化降级。
- WebSocket 候选协议按 `stream + stream_id` 判断缺口，按 `event_id` 去重；批准前不得冻结实现。
- 实时重连必须先获取权威快照，再按授权增量补偿；不得双向合并本地旧投影。
- SSE 用于生成任务进度；游戏权威状态走 WebSocket，不得混用。

### 6.3 决策状态

`Accepted` 决策：

- 资源 ID 为纯 26 字符 ULID，不使用类型前缀。
- 图片生成与 AI 复盘点评为 P1 异步增强，不得反向阻塞 P0。
- 正式 PRD 路径固定为 `docs/MysteryVerse-PRD.md`。

`Proposed` 决策，未获负责人确认前不得作为实现契约：

- Memory 元数据由 Go 唯一权威写入，Python 只提交 MemoryCandidate。
- WebSocket 统一 Envelope、stream 和 seq 语义。
- `event_seq` 与 `state_version` 分离语义。
- 局内暂停/恢复使用独立 G09/G10。
- 等待房聊天使用 `room_messages`、`room_chat` 与独立 `chat_seq`。
- 基础运营审核后台是公开 MVP 发布门禁。
- PlayerContext 在 MVP 中每次从授权可见投影重建。

完整状态和任务门禁见 `docs/architecture/DECISION-BASELINE.md`。

决策冻结不等于契约实现完成。Go/Python 内部 OpenAPI、JSON Schema、WS fixture、
数据库迁移和生成类型仍需按 TASKS 创建并通过 GATE。

## 7. 开发约束

- 不允许把业务逻辑全部写在 HTTP Handler 中。
- 不允许把游戏规则、最终权限判断、幂等判断或胜负判定放到前端。
- 不允许让 LLM 直接修改权威游戏状态。
- 不允许使用 Redis 作为唯一权威数据源。
- 不允许为了架构复杂度过早引入微服务集群。
- 不允许无故增加技术栈和依赖。
- 不允许用 `any`、强制类型断言或关闭严格校验绕过领域 DTO。
- 不允许在业务组件中散落魔法颜色、字号、阴影或无 Token 的视觉值。
- 不允许在日志中记录 D2-D4 正文、Prompt、Token、Cookie、密钥或完整私聊。
- 禁止提交真实密钥、数据库凭据、模型密钥、证书或生产 `.env`。
- 不在应用启动时自动修改生产数据库 Schema；迁移必须版本化并可回滚。
- 事务内不得调用 LLM、对象存储或其他外部 HTTP 服务。
- 大对象先写对象存储，再在事务中登记引用；失败对象由生命周期任务回收。
- 发布、结算、角色分配、线索争抢和投票必须由数据库约束与事务保护。
- 不得删除、覆盖或回退不是本任务产生的用户修改。

## 8. 代码质量

所有任务必须满足：

- 类型安全。
- 清晰的模块边界。
- 可测试性。
- 完整错误处理。
- 日志与可观测性。
- 安全性。
- 必要的代码注释。

分层要求：

- Go `domain` 不依赖 Chi、WebSocket、Redis、MySQL 或对象存储。
- Go `application` 负责编排事务、权限和领域服务。
- Go `adapter` 实现数据库、Redis、对象存储和 AI HTTP 端口。
- Go `transport` 只做协议转换、认证上下文提取和错误映射。
- Python API routes 只做协议层；业务在 workflow、agent、schema、guard 和 provider 层。
- Python 访问 Go 的内部客户端集中放在 `providers/go_core`，不得散落业务模块。
- React 页面不直接消费数据库 Entity；必须经过生成 DTO、API adapter、view model。
- 跨模块调用通过应用接口，不直接引用对方内部 Repository。

错误处理必须：

- 使用统一错误码和错误 envelope。
- 区分校验、权限、冲突、依赖、AI 超时、安全拦截和工作流失败。
- 保留可追踪 `request_id`/`trace_id`，但不向客户端泄漏堆栈、SQL、Prompt 或供应商原文。
- 超时、取消、重试、版本冲突和降级都有明确行为。

## 9. 测试与质量门禁

所有重要业务逻辑必须有测试。测试至少覆盖对应风险：

- Go：规则表驱动测试、属性测试、事务并发、幂等、快照恢复、事件重放、权限投影和 WebSocket 协议。
- Python：Pydantic/JSON Schema、节点单元、Graph 分支、Checkpoint、Interrupt、Repair、取消、恢复、Provider fake、注入与泄漏测试。
- 前端：Vitest + React Testing Library、关键状态矩阵、Playwright、移动端、键盘、Focus、Reduced Motion 和实时重连。
- 跨语言：OpenAPI/JSON Schema 漂移、WS fixture、GamePackage fixture、Go 编译 Python 产物的合同测试。
- 安全：D2-D4 越权、私聊、角色本、狼人阵营、Prompt Injection、对象存储旁路和分享泄漏。
- 数据：迁移正反向、影子库和数据库不变量。
- 性能：游戏 P95、WebSocket 扇出、重连恢复、AI 超时和 500 活跃房间目标。

测试失败不得通过删除用例、跳过测试、放宽断言或修改设计约束来规避。

## 10. 文档维护

| 变更类型 | 必须同步检查 |
| --- | --- |
| API、WebSocket、SSE 或错误码 | `docs/API_DESIGN.md` |
| 表、字段、索引、约束或迁移 | `docs/DATABASE_DESIGN.md` |
| 技术边界、目录、协议或基础设施 | `docs/TECH_DESIGN.md` |
| 页面、组件、状态、Token 或交互 | `docs/UI_DESIGN.md` |
| 任务状态、依赖、范围或验收 | `docs/TASKS.md` |
| 设计冲突、架构风险或待决事项 | `docs/DESIGN_REVIEW.md`；需要时另行建立经批准的技术决策记录 |

规则：

- 修改 API 时同步检查 `docs/API_DESIGN.md`。
- 修改数据库结构时同步检查 `docs/DATABASE_DESIGN.md`。
- 修改任务状态时更新 `docs/TASKS.md`。
- 发现设计问题时记录到 `docs/DESIGN_REVIEW.md` 或独立技术决策记录。
- 不允许为了让代码通过测试而随意修改设计约束。
- 文档只应修改与当前任务相关的内容，不得顺手重写已完成设计。

## 11. 任务执行协议

后续每次开发必须按以下顺序执行：

1. 阅读根 `AGENTS.md` 和相关设计文档。
2. 阅读 `docs/TASKS.md` 中的目标任务、前置依赖、涉及文件、测试和验收标准。
3. 检查 `git status`，识别未提交及未跟踪内容，不覆盖用户已有修改。
4. 明确任务范围、前置门禁、契约和数据依赖。
5. 一次只领取一个任务；开始后将任务状态改为 `DOING`。
6. 先更新或冻结必要契约、Schema、迁移，再实现业务。
7. 编写实现，并补充成功、失败、权限、并发、重试和降级测试。
8. 执行任务要求的测试、Lint、类型检查、Schema 漂移和相关集成测试。
9. 修复失败项；不得用跳过测试或放宽约束规避问题。
10. 更新 `docs/TASKS.md` 状态、实际变更、测试命令、结果、证据和剩余风险。

只有全局 DoD 和任务 DoD 同时满足时，任务才能标记 `DONE`。

TASKS 状态维护至少包含：

```text
状态：DONE
实际变更：<文件列表>
测试命令：<命令>
测试结果：<通过/失败摘要>
验收证据：<报告、截图、trace、fixture 或链接>
剩余风险：<无或风险编号>
```

BLOCKED、DEFERRED 或未决契约不得通过假设继续实现。发现新冲突时先登记并请求决策。

## 12. 当前基线与开工门禁

- 当前仓库已具备 Monorepo、Contracts、数据库迁移、数据访问平台、用户认证和等待房 REST 业务代码。后续任务不得把这些既有实现视为空仓库并重新生成。
- 不得仅因设计文档存在就初始化项目；工程初始化必须由明确的 TASKS 任务或用户指令触发。
- 当前 ADR 状态仍为 3 项 Accepted、7 项 Proposed；DQ-18/DQ-19 已获负责人批准
  并完成测试级实现，DQ-20 至 DQ-22 仍为 Proposed。只有 Accepted ADR 和已批准
  DQ 范围形成强制基线。
- `GATE-API`、`GATE-WS`、`GATE-GAMESTATE`、`GATE-ADMIN` 只有在契约、生成物、迁移和测试实际存在后才能通过。
- Proposed 决策获得负责人确认前，依赖它们的实时、事件、后台、Memory 和 AI 上下文任务不得冻结契约。
- 当前 `docs/TASKS.md` 中 `P1-DB-006`、`P1-DB-007`、`P1-DB-008`、`P1-DB-009`、`P1-GO-009` 等仍受 Proposed ADR 或契约门禁限制；Phase 5、6、7 的多项任务为 `DEFERRED`。执行前必须重新读取实时状态。
- Task 5 已实现 email + 密码认证和等待房 REST，但没有真实 `room_chat` WebSocket 广播、邀请创建/解析端点和 WebSocket ticket；不得把现有 REST 支持描述为 `GATE-WS` 已通过。
- Task 6 已实现候选 Go 游戏引擎、R09、G01/G02/G06、G09/G10、M01-M05、M07/M08、
  原子开局、角色/线索权限、投票和结算；ADR-004/ADR-005 仍为 Proposed，
  因此不得把当前事件版本、暂停恢复或实时游戏协议描述为已冻结。
- Task 7 已实现剧本 Draft、结构化对象 CRUD、版本深拷贝、ScriptValidator、
  私有/邀请发布、下架和只读发布版本 Loader。章节/结局/资产/发布记录有最小数据承载；
  但这些结构补充不批准条件 DSL、投票策略、ViewerManifest 或 SecretPayload AAD。
  `channel=public` 在审核 ADR/流程实现前返回 `MODERATION_REQUIRED`，
  `private_profile_ref` / `private_text_ref` 在 AAD 批准前不得写入。
- Task 8 已实现 Python 通用 LangGraph Agent Runtime、Mock/OpenAI-compatible
  Provider、注册式 Tool Registry、授权 Context Builder、非权威短期 Memory 端口、
  Trace/脱敏、内部 `/internal/v1/ai/agent-runs` 和最小 Go InternalAIClient。
  当时只有 `demo` 任务级输出 Schema；AI DM/NPC/生成业务、外部 MCP、`mv_ai`
  trace 持久化与 LangGraph checkpoint 仍未实现，不得描述为 GATE-AI 已通过。
- Task 9 已补充版本化输出 Schema Registry、可选 HMAC/timestamp/nonce 请求完整性
  原型、双签名密钥轮换窗口、D3/D4 默认上下文阻断、Tool trust/确认/结果上限和
  进程内 Run 元数据原型。GATE-AI 仍未通过；外部 MCP、Checkpoint 持久化、
  真实 Provider 政策和业务 Schema 仍以
  `docs/architecture/gate-ai-checklist.md` 与 DQ-18 至 DQ-22 为准。
- Task 10 已完成 DQ-18 至 DQ-22 候选设计。Task 11 经负责人 2026-11-25 授权后，
  DQ-18 已实现正式 Contracts Registry、兼容性基线、Active Schema 和 Demo 输出
  绑定；DQ-19 已实现 `mv-hmac-sha256-v1`、Key ID/Key Provider、nonce Store、
  scope ACL、安全审计、FastAPI TLS 配置和 Go mTLS Client。真实 CA、KMS、Redis
  多实例和生产 mTLS 部署仍为 `NOT VERIFIED`。
- DQ-20 至 DQ-22、ADR-001/004/005/008 仍未被 Task 11 授权批准。外部 MCP、
  Run/Checkpoint、真实 Provider、Memory、正式事件版本、暂停恢复和持久
  PlayerContext 继续保持门禁，见 `docs/architecture/task-11-approval-blockers.md`。
- 开始任何 TASKS 任务前，应先完成其契约缺口和依赖检查；不得把 BLOCKED、DEFERRED 或 P1 可选能力反向当作无争议的 P0 前置。
