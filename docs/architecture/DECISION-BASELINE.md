# MysteryVerse 技术决策基线

> 状态：Conditional Baseline
> 复核日期：2026-11-25
> 任务：Task 1 设计审查问题收敛与开发基线冻结
> 批准规则：只有 `Accepted` 决策可作为强制实现依据；`Proposed` 必须等待负责人确认。

## 1. Task 0 交付复核

| 检查项 | 结论 |
| --- | --- |
| `AGENTS.md` | 存在，包含项目定位、架构、服务边界、约束、测试和任务协议 |
| 正式 PRD | 实际为 `docs/MysteryVerse-PRD.md`，未发现重复 PRD |
| 设计文档 | TECH、DB、API、UI、TASKS、DESIGN_REVIEW 均存在 |
| 业务代码 | 不存在 |
| 前后端工程 | 未初始化 |
| Git | `master` 无提交，仓库内容全部未跟踪 |
| TASKS | 结构可执行，但部分 P0 依赖 Proposed 决策，必须设置确认门禁 |
| DESIGN_REVIEW | 历史问题仍在文件中；ADR 附录已记录候选收敛，但批准状态需以本文件为准 |

## 2. 决策状态总表

| ID | 决策 | 状态 | 依据与原因 | 开发门禁 |
| --- | --- | --- | --- | --- |
| ADR-001 | Go 唯一写 Memory，Python 提交 Candidate | Proposed | 原 TECH、DB、TASKS 存在写权冲突；修复未获负责人确认 | 不阻塞剧本杀主闭环；阻塞 Phase 5/NPC/RPG Memory |
| ADR-002 | 统一 WebSocket Envelope | Proposed | 原 TECH/API 字段与序号语义冲突；新协议需确认 | 阻塞 GATE-WS、实时联调和 `room_chat` |
| ADR-003 | 纯 26 字符 ULID | Accepted | DB/API 已明确 26 字符 ULID；原示例是文档缺陷，已统一 | 所有 Schema、迁移和 DTO 必须遵守 |
| ADR-004 | `event_seq` 与 `state_version` 分离 | Proposed | 原 DB/API/TECH 对聊天和 DM 的版本语义冲突 | 阻塞事件模型、GATE-GAMESTATE 和相关实时测试 |
| ADR-005 | 独立暂停/恢复控制命令 | Proposed | 原 API 将 pause/resume 放在 room host-actions | 阻塞游戏控制命令和暂停恢复 E2E |
| ADR-006 | 等待房聊天进入 MVP | Proposed | PRD 有页面需求，但原 API/DB/WS 未定义；具体隔离方案待确认 | 阻塞 R13/R14、`room_messages` 和 P1-GO-009 |
| ADR-007 | 基础运营后台为公开 MVP 门禁 | Proposed | PRD 有后台需求，但内测/公开范围及最小权限方案待确认 | 阻塞公开 MVP，不阻塞私有剧本杀内测 |
| ADR-008 | PlayerContext 每次重建、不持久化 | Proposed | 原始 PRD 定义 PlayerContext 实体；MVP 临时方案待确认 | 阻塞 AI DM/NPC 上下文实现；不阻塞纯规则引擎 |
| ADR-009 | 图片与 AI 点评为 P1 异步增强 | Accepted | PRD/TECH 已明确可关闭、可降级；原 TASKS 属依赖倒置缺陷 | P1 可后置，不得阻塞 P0 |
| ADR-010 | PRD 固定为 `docs/MysteryVerse-PRD.md` | Accepted | 文件实际存在且各文档已引用正确路径 | 禁止创建重复 `docs/PRD.md` |

## 3. 已冻结技术基线

以下内容已经具备明确设计与无冲突文档依据，可直接作为开发约束：

### 3.1 资源 ID

- 业务资源 ID 为纯 26 字符 ULID。
- 数据库主键为 `CHAR(26)` ASCII binary collation。
- API、WebSocket、前端生成类型、日志资源值不得添加 `game_`、`room_` 等前缀。
- 类型前缀仅可作为 UI 展示标签。

### 3.2 AI 异步增强范围

- 图片生成和 AI 复盘点评为 P1。
- 失败、超时或关闭不得阻塞游戏核心流程。
- P0 任务不得硬依赖 P1 任务。

### 3.3 PRD 路径

- 正式 PRD 为 `docs/MysteryVerse-PRD.md`。
- 不创建 `docs/PRD.md`，不维护两份 PRD。

## 4. 条件基线

以下内容描述当前唯一候选方案，但在获批准前不得成为强制契约：

### 4.1 核心服务边界

- Go 是用户、房间、游戏状态、角色、线索权限、投票、结算、事件、Memory 元数据和运营处置的权威写入者。
- Python 只负责生成、结构化、校验、DM/NPC、MemoryCandidate、复盘和模型运行时。
- React 只消费 Public/Viewer/Private DTO，不执行权限、规则或幂等裁决。
- MySQL 是业务权威数据源；Redis 和本地内存不是权威；对象存储保存不可变大对象。

### 4.2 数据库权威归属

| 数据 | 权威归属 |
| --- | --- |
| 用户、内容、房间、游戏、投票、结算 | Go `mv_core` |
| GameEvent、RoomEvent、RoomMessage | Go |
| NPC Memory 元数据与加密摘要 | Go `mv_core` + SecretService |
| Prompt、Run、Usage、Checkpoint、MemoryCandidate | Python `mv_ai` |
| 运营、举报、处罚、审计 | Go `mv_core` |

### 4.3 WebSocket 候选规范

```text
event_id, event_type, stream, stream_id, seq,
schema_version, game_id?, timestamp, payload
```

- `stream` 候选值：`system`、`room`、`room_chat`、`game`、`ai_progress`、`ai_stream`。
- `seq` 作用域为 `stream + stream_id`。
- 游戏事件必须带 `game_id`；等待房事件不得伪造 `game_id`。
- 客户端按 `event_id` 去重，按 `seq` 检测缺口。

### 4.4 事件与状态版本候选规范

- 所有权威游戏事件递增 `event_seq`。
- 聊天、DM 和正式叙事只递增 `event_seq`。
- 投票、阶段、暂停、恢复、线索、技能和结算递增 `event_seq + state_version`。
- 事件可见性由服务端 ACL 和投影决定，统一存储不代表统一公开。

### 4.5 Agent 权限候选规范

- Go 生成授权知识切片。
- Python 只能在切片内排序、摘要、裁剪和生成候选。
- 同步 AI 回合不得形成 Python 回调 Go 再调用 Python 的嵌套环。
- PlayerContext 在批准前只能作为待评审方案，不得直接实现持久化。

## 5. 未决问题与阻断关系

| 编号 | 未决决策 | 影响 |
| --- | --- | --- |
| DQ-01 | MemoryCandidate 最终写权与版本策略 | Phase 5、NPC/RPG Memory |
| DQ-02 | WebSocket Envelope 与 stream/seq 规范 | 所有实时联调、GATE-WS |
| DQ-03 | Event/State Version 对消息事件的精确语义 | 事件模型、重连、回放、GATE-GAMESTATE |
| DQ-04 | 暂停/恢复控制端点与权限 | Game Runtime、Timer、API |
| DQ-05 | 等待房聊天 MVP 接口与保留策略 | `room_messages`、R13/R14、等待房 UI |
| DQ-06 | 内测/公开 MVP 后台门禁边界 | Admin API、公开上线 |
| DQ-07 | PlayerContext 是否采用临时重建方案 | AI DM/NPC、Context Resolve |

其他工程未决项：

| 编号 | 问题 | 影响 |
| --- | --- | --- |
| EQ-01 | LangGraph TaskNode/Validation 子图拓扑 | P2-PY-017 与 UI 进度映射 |
| EQ-02 | DerivationPlan/TruthWitness 形式化证明 | 发布校验与无解案件门禁 |
| EQ-03 | 父任务、节点、Provider 的 deadline 预算 | P2-PY-016 与 AI 降级 |
| EQ-04 | 候选 Diff、Artifact 查询和接受/拒绝 API | 编辑器人工审核闭环 |
| EQ-05 | 大厅搜索、评分读模型、通知、成就和完整历史 | 公开规模与页面完整性 |
| EQ-06 | 大粒度任务和需求编号规范 | 任务可执行性与追踪 |

AI Runtime 新未决项：

批准证据：负责人于 2026-11-25 在明确要求 DQ-18/DQ-19 批准后回复“开始执行吧”。
该授权只覆盖 Registry 与内部服务安全基础，不延伸到 DQ-20 至 DQ-22。

| 编号 | 未决决策 | 当前状态 | 影响 |
| --- | --- | --- | --- |
| DQ-18 | 任务级输出 Schema 与正式版本治理 | Approved / Implemented / Verified | 正式 Registry、兼容性、Active 与 Runtime 输出绑定已落地 |
| DQ-19 | mTLS、请求签名、nonce 存储和密钥轮换 | Approved / Implemented / Test-Verified；生产基础设施 Not Verified | Go/Python 内部通信基础已落地；真实 CA/KMS/Redis 多实例待部署验证 |
| DQ-20 | MCP Server 信任、工具发现和审计 | Proposed | 外部工具接入 |
| DQ-21 | Run/Checkpoint 存储、恢复与保留 | Approved / Implemented / Hosted CI Verified；Production Not Verified | CI Run `45803587621`：Integration、Multi-Worker、Recovery/Cleanup、Backup/Restore 全部通过 |
| DQ-22 | 真实供应商、数据边界、预算和降级政策 | Proposed | 真实模型上线 |

详细候选方案、依赖图和验收要求：

- `docs/architecture/dq-18-22-ai-runtime-design.md`
- `docs/architecture/ai-runtime-schema-inventory.md`
- `docs/architecture/gate-ai-checklist.md`
- `docs/architecture/task-11-approval-blockers.md`
- `docs/architecture/contracts-registry.md`
- `docs/architecture/task-11-report.md`

## 6. 当前可执行与不可执行范围

可以开始：

- `P0-INF-001` Monorepo 与工程目录。
- 26 字符 ULID、PRD 路径和 P1 异步增强相关基础设施约束。
- DQ-18 正式 Schema Registry、兼容性检查、Active 绑定和生成流水线。
- DQ-19 已实现的 HMAC、Key ID、nonce 抽象、ACL scope、审计与 TLS 客户端配置。
- Go/Python/Frontend 基础平台的接口抽象，但不得冻结 Proposed 契约。

必须等待负责人确认：

- WebSocket Envelope、stream 和重连协议。
- `game_events` 与消息事件的最终版本语义。
- 暂停/恢复 API。
- 等待房聊天数据库与 API。
- 运营后台公开 MVP 范围。
- PlayerContext 临时重建方案。
- Memory 权威写入细节。
- DQ-20 至 DQ-22 对应外部 MCP、Run/Checkpoint 和真实 Provider 能力。

## 7. 批准后的文档动作

负责人确认某项 Proposed 决策后：

1. 将 ADR 状态改为 `Accepted`。
2. 更新本文件状态表和门禁。
3. 更新 `AGENTS.md`、`TASKS.md` 和对应 API/DB/TECH/UI 文档。
4. 解除对应任务门禁。
5. 若批准内容与候选方案不同，先更新 ADR 和契约，再开始实现。

## 8. 结论

Task 11 后形成新的条件开发基线：ADR 仍为 3 项 Accepted、7 项 Proposed；
DQ-18/DQ-19 已批准并完成测试级实现验证，DQ-20 至 DQ-22 仍为 Proposed。
核心闭环的实现方向已清晰，但在 7 项 Proposed 获得负责人确认前，不得将当前候选
方案当作正式冻结契约，也不得开始依赖这些契约的跨服务或实时联调。

