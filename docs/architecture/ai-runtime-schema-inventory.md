# AI Runtime Schema Inventory

> 日期：2026-11-23
> 状态：APPROVED / IMPLEMENTED / VERIFIED（DQ-18）
> 正式 Schema 源：`packages/contracts/json-schema/mysteryverse.schema.json`
> Registry：`packages/contracts/registry/schema-registry.json`

## 1. 分类清单

| 类别 | 当前 Schema / 类型 | 当前位置 | 状态 | 正式 Contracts 建议 |
| --- | --- | --- | --- | --- |
| Agent 同步请求/响应 | `AgentRequest`、`AgentResponse` | `packages/contracts` | Mixed / 部分受 ADR 阻塞 | 纳入 Registry；字段能力按 ADR 状态使用 |
| Agent Context | `AgentExecutionContext`、`AgentContextMessage`、`PlayerContext` | `packages/contracts` | Proposed | 纳入，受 ADR-008 门禁 |
| Agent 输出身份/校验 | `output_schema_key/version`、`AgentOutputValidationIssue`、`DemoAgentOutput` | `packages/contracts` | DQ-18 Approved | 纳入；Demo 绑定 Active |
| Agent Trace / Tool Call 摘要 | `AgentTraceStep`、`AgentToolCall`、`AgentExecutionStatus` | `packages/contracts` | Proposed | 纳入对外 Run 摘要 |
| API/错误/分页 | `ApiSuccessResponse`、`ApiErrorResponse`、`BusinessErrorCode` | `packages/contracts` | Mixed | 纳入 |
| Game/Room/AI 事件 | `GameEventType` 及事件 payload | `packages/contracts` | Proposed | 纳入，受 ADR-004 门禁 |
| WebSocket | `WebSocketEnvelope` | `packages/contracts` | Proposed | 纳入，受 ADR-002/004 门禁 |
| MemoryCandidate | `MemoryCandidate` | `packages/contracts` | Proposed | 纳入，受 ADR-001 门禁 |
| ScriptGeneration | `ScriptGenerationRequest/Result` | `packages/contracts` | Proposed | 纳入，受 DQ-08 至 DQ-12 门禁 |
| Provider Request/Response | Python `ModelRequest`、`ProviderResult`、`ProviderUsage` | `ai_service/providers/models.py` | Internal | 不进入公共 Contracts，除非未来跨服务直接调用 |
| Provider 配置 | Python `Settings` | `ai_service/config.py` | Internal Secret Config | 不进入 | 
| Tool 参数 Schema | 各 `ToolSpec.args_model` Pydantic Schema | `ai_service/tools/` | Internal/Per-tool | 本地工具可内部；跨服务/MCP Tool 需正式 Schema |
| Tool 执行结果 | `ToolResult`、`ToolExecution` | `ai_service/tools/models.py` | Internal | 只在授权审计/跨服务桥接时进入 Contracts |
| Tool 信任策略 | `ToolSpec` trust/source/permission | `ai_service/tools/models.py` | Candidate Internal | Pydantic 内部模型，策略批准后可能生成 Contracts |
| MCP Discovery/Tool Schema | 尚无 | - | NOT_IMPLEMENTED | DQ-20 批准后设计 |
| Run 生命周期元数据 | `AgentRunRecord` | `ai_service/agent_runtime/run_store.py` | Candidate Internal | Go 查询时纳入稳定 Run DTO |
| Checkpoint 引用 | `CheckpointReference` | `ai_service/agent_runtime/run_store.py` | Candidate Internal | 引用元数据可纳入；LangGraph State 不纳入 |
| LangGraph 内部 State | `AgentGraphState` | `ai_service/agent_runtime/state.py` | Private Implementation | 不纳入 |
| Agent 模型 action | `ModelAction` | `ai_service/agent_runtime/outputs.py` | Private Implementation | 不纳入 |
| 输出 Schema Registry | `SchemaRecord`、`RuntimeSchemaBinding`、`OutputSchemaRegistration` | Contracts Registry + Python Runtime | Approved / Implemented / Verified | Registry 为治理权威，Runtime 只绑定 Active schema |
| 内部认证 header/canonical | HMAC header 与签名规范 | `docs/api/ai-internal-api.md`；security modules | DQ-19 Approved / Test-Verified | `mv-hmac-sha256-v1` internal security profile |
| 持久化 DDL | `mv_ai.agent_*` / runtime audit / cleanup | `DATABASE_DESIGN.md`；`00009_ai_runtime_persistence.sql` | Migrated in source / DB execution NOT VERIFIED | 内部 Schema，不进入公共 Contracts |

## 2. Contracts 纳入规则

必须进入 Contracts：

- Go、Python、React 之间传输的请求、响应、事件、错误和 Run 摘要。
- 已批准的任务级输出 Schema 和 tool invocation 跨服务协议。
- 对外可查询的 Run、Checkpoint 引用和审计摘要。
- OpenAPI/WS/SSE 共同引用的公共类型。

不进入 Contracts：

- Provider 内部 request/response、供应商 SDK 结构。
- LangGraph 内部 state、node checkpoint 二进制/blob。
- Prompt 拼接内部对象、模型 action。
- Secret、API Key、原始 prompt/私密正文。
- 单一进程内的缓存和锁对象。

条件进入：

- MCP Tool 元数据/参数/结果：只有 DQ-20 批准并经 Go/Python 边界验证后。
- Checkpoint 元数据：引用和摘要可进入；内容格式不进入。
- 工具审计：共享字段进入 Contracts，Server 原始 payload 留在受限存储。

## 3. Versioning Policy

1. 跨服务 Schema 使用显式 `schema_version` 或 `SemVer`。
2. 任务级输出使用 `schema_key + schema_version`，模型不能自选。
3. 非兼容变更新增 Major；可追加字段使用 Minor，但客户端必须声明未知字段策略。
4. Schema 生命周期：`draft → active → deprecated → retired`。
5. Active 版本由服务端配置/注册表决定，不由请求方或模型自由选择。
6. 历史 Run 保留实际 key/version；重放需要对应版本仍可用或迁移报告。
7. 多版本并行运行只允许显式注册的兼容窗口，不允许无版本 fallback。
8. 生成物漂移、三语言测试和 fixture 是合并门禁。

## 4. Compatibility Policy

- 新增可选字段：兼容。
- 删除/重命名、改变必填、语义或枚举含义：不兼容。
- 枚举只增不改；消费方必须拒绝未知关键枚举或显式降级。
- `additionalProperties=false` 用于安全 DTO；扩展走正式字段。
- 旧 Run 无法解析时不自动降级到“忽略失败”，必须标记 `CHECKPOINT_UNAVAILABLE`。

## 5. Active / Deprecation / Rollback

- Active：每个 task_type 可有一个默认 key/version，但请求可显式选择 allowlist 版本。
- Deprecation：保留兼容窗口并记录替代版本；新 Run 不再默认进入 deprecated。
- Rollback：回滚只切换默认 active 指针，不删除旧 Schema/Run 引用。
- Retired：只在无活跃 Run 且保留期结束后禁用新执行；历史审计按保留策略处理。

## 6. Resolved Questions

- 业务 Schema 全部由 `packages/contracts` 生成 Registry。
- 被历史 Event/Run 引用的 Schema 在其引用生命周期内保留；Retired 仍只读。
- 请求可显式选择 allowlist 版本；默认 Active 只能来自服务端 Registry。
- Run/Checkpoint DTO 仍由 DQ-21 阻塞，不能由本 Registry 代批。
- MCP Tool Schema 仍由 DQ-20 阻塞。

完整兼容性、Replay 和生命周期规则见
`docs/architecture/contracts-registry.md`。
