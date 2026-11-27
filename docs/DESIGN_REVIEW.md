# MysteryVerse 全局架构与设计文档一致性审查

> 审查日期：2026-11-20
> 审查范围：`MysteryVerse-PRD.md`、`TECH_DESIGN.md`、`DATABASE_DESIGN.md`、`API_DESIGN.md`、`UI_DESIGN.md`、`TASKS.md`
> 审查性质：独立架构、Go/Python 边界、LangGraph、Memory、游戏引擎、数据库/API、UI 与任务执行性审查
> 审查结论：**有条件通过，暂不批准直接按当前文档冻结全部 P0 实施契约**
> 总体评分：**7.1 / 10**

---

## 0. 审查说明

### 0.1 路径基线

用户指定的 `docs/PRD.md` 在仓库中不存在。仓库实际 PRD 文件为 `docs/MysteryVerse-PRD.md`。本次审查完整读取该文件，并同时读取以下五份文档：

- `docs/TECH_DESIGN.md`
- `docs/DATABASE_DESIGN.md`
- `docs/API_DESIGN.md`
- `docs/UI_DESIGN.md`
- `docs/TASKS.md`

本次只做文档审查，不修改任何现有需求、技术、数据库、API、UI 或任务文档。

### 0.2 严重程度定义

| 严重程度 | 定义 |
| --- | --- |
| Critical | 违反不可突破的架构红线，可能造成权限旁路、权威状态被非授权服务修改或无法审计的数据污染 |
| High | 会阻断 MVP 契约冻结、核心闭环实现、关键测试门禁或公开上线条件 |
| Medium | 会造成明显返工、性能问题、需求覆盖缺口或跨团队理解偏差，但存在可控降级或后置路径 |
| Low | 主要影响文档一致性、术语清晰度或实现细节，不改变核心架构结论 |

### 0.3 文档规模与可执行性概览

| 项目 | 当前规模 |
| --- | --- |
| PRD | 3859 行 |
| TECH_DESIGN | 3532 行 |
| DATABASE_DESIGN | 3759 行，约 79 个表级设计小节 |
| API_DESIGN | 2392 行，约 84 个公共接口契约小节 |
| UI_DESIGN | 3896 行，约 20 个页面设计小节 |
| TASKS | 3241 行，135 个任务 |
| TASKS 状态 | 100 个 P0 TODO；9 个 P1 TODO；2 个 P1 BLOCKED；25 个 DEFERRED |

任务文档并非“少量骨架任务”，而是完整实施计划。当前主要问题不在任务数量，而在契约仍有未决冲突、P0 标签过宽、部分 P0 任务依赖 BLOCKED 或 DEFERRED 前置。

---

## 1. 总体结论与评分

### 1.1 总体评价

架构主方向是正确的：

- Go 作为权威游戏状态唯一写入者。
- Python 只负责 AI 生成、理解、表达和候选数据。
- 采用模块化单体 + 进程隔离，而非过早拆分大量微服务。
- 游戏状态使用 MySQL 当前状态 + 不可变事件 + 快照恢复。
- Redis 明确为缓存、租约、实时分发和任务流，不承担游戏权威。
- LangGraph 具备 State、条件边、并行汇合、Repair 上限、HumanReview interrupt、Checkpoint 和结构化输出。
- 权限、DTO 投影、幂等、事件序号、重连、失败回滚和发布原子性均有较完整设计。
- UI 设计系统、三模式差异、AI 工作流可视化、状态矩阵、动效契约和移动端适配明显高于一般设计基线。

但文档之间尚未达到“可直接冻结为唯一实施契约”的程度。当前最需要解决的是：

1. Memory 的 Go/Python 写权边界自相矛盾。
2. WebSocket 与内部 AI 服务协议还没有唯一权威版本。
3. 聊天、DM 等非规则事件与 `state_version`、`event_seq` 的关系未定义清楚。
4. LangGraph 拓扑与 TASKS、UI 中的节点集合不一致。
5. 可推导性与逻辑矛盾自检缺少可证明算法。
6. 若干关键数据/API 缺口尚未关闭，包括邀请 token、候选 Diff、后台 API。
7. TASKS 存在 BLOCKED、DEFERRED 和可选功能反向阻塞 P0 任务的依赖问题。

### 1.2 分项评分

| 维度 | 分数 | 结论 |
| --- | ---: | --- |
| Go 与 Python 架构边界 | 7.0 | 主边界正确；Memory 写权和上下文编译职责存在冲突 |
| LangGraph 工作流 | 7.5 | State、Checkpoint、Repair、HumanReview 设计较完整；节点拓扑与自检证明不足 |
| Memory | 6.5 | 作用域、来源、权威优先级较好；写权、版本和 PlayerContext 缺口明显 |
| 游戏引擎 | 8.0 | 三模式独立、单写、幂等、快照、回放较强；事件版本与暂停控制仍需修正 |
| 数据库与 API | 6.5 | 数据模型成熟；存在缺表、ID 口径、候选 Diff/后台 API 和读模型缺口 |
| UI | 8.5 | 页面覆盖、Tailwind 系统、三模式体验、状态和响应式设计较完整；受后端契约缺口影响 |
| TASKS | 5.5 | 任务有验收与测试，但依赖门禁、粒度和优先级范围存在明显执行风险 |
| 跨文档一致性 | 6.5 | 大量术语和契约仍处于“已提出修正但源文档未收敛”状态 |

加权总分：**7.1 / 10**。

### 1.3 建议评审状态

| 事项 | 结论 |
| --- | --- |
| Phase 0 工程骨架 | 可在 `P0-DOC-001` 完成并登记本报告问题后谨慎启动 |
| GATE-API / GATE-WS | 暂不得通过 |
| GATE-GAMESTATE | 暂不得通过 |
| 剧本杀 MVP 契约冻结 | 条件不满足，先修复全部 MVP 阻塞项 |
| NPC/RPG/Memory 完整实现 | 条件不满足，先修复 Memory 权威边界 |
| 公开 MVP 上线 | 条件不满足，需补齐运营审核 API 与端到端测试任务 |

---

## 2. 问题总表

| ID | 严重程度 | 问题 | 是否阻塞 MVP |
| --- | --- | --- | --- |
| ARC-001 | Critical | Memory 长期记录写入权与 Go 权威边界矛盾 | 否（剧本杀主闭环）；是（NPC/RPG） |
| ARC-002 | High | WebSocket 公共信封在 TECH 与 API 之间不一致 | 是 |
| ARC-003 | High | Go/Python 内部 AI 协议缺少字段级完整契约和回调防环规则 | 是 |
| ARC-004 | Medium | 上下文编译职责在 Go 与 Python 间重叠 | 否 |
| ARC-005 | Medium | `Project` / `Script` 规范命名尚未在 TECH 中收敛 | 否 |
| LNG-001 | High | LangGraph 拓扑、TASKS 节点和 UI 节点映射不一致 | 是 |
| LNG-002 | High | 自检无法证明线索可推导、真相唯一和阻断矛盾零漏检 | 是 |
| LNG-003 | Medium | 内容生成总超时与阶段超时冲突 | 否 |
| MEM-001 | Medium | PRD `PlayerContext` 在 TECH、DB、TASKS 中未落地 | 否 |
| MEM-002 | Medium | 长期记忆缺少版本字段与并发约束 | 否 |
| ENG-001 | High | 聊天、DM 等事件与 `state_version` / `event_seq` 语义冲突 | 是 |
| ENG-002 | High | 局内暂停/恢复只依赖房间序号的契约不足 | 是 |
| DATA-001 | High | 邀请 token 表缺失 | 是 |
| DATA-002 | Medium | 26 字符 ULID 与带前缀示例的长度口径冲突 | 是（契约冻结前） |
| DATA-003 | Medium | 大厅搜索、玩家/时长筛选和评分读模型缺索引或聚合结构 | 否（小规模 MVP）；是（公开规模） |
| API-001 | High | 运营审核后台缺少公共后台 API、任务和 UI 接入覆盖 | 是（公开 MVP） |
| API-002 | High | 编辑器候选 Diff、Artifact 查询和接受/拒绝接口缺失 | 是 |
| API-003 | High | 等待房聊天仍无 API，但 PRD/UI 将其作为页面能力 | 是（若保留该需求） |
| API-004 | Medium | 完整历史、通知、成就和首页聚合的 DB/API 契约不完整 | 否（核心对局）；是（页面完整性） |
| API-005 | Medium | UI 使用 API 未定义的错误码语义 | 否 |
| API-006 | Medium | 客户端消息允许 `system_note`，存在伪造系统消息风险 | 否（实现前必须修正） |
| TASK-001 | High | MVP E2E 任务依赖 DEFERRED 的 RPG 页面 | 是 |
| TASK-002 | High | BLOCKED 的宽范围契约任务阻塞多个 P0 页面任务 | 是 |
| TASK-003 | High | 可选 P1 功能反向硬阻塞 P0 任务 | 是 |
| TASK-004 | Medium | 多个任务粒度过大，且 P0 TODO 数量失控 | 否（但显著影响交付） |
| TASK-005 | Medium | P5 Memory 被标为 P0，但未纳入 MVP 必做任务集合 | 否（需范围签署） |
| TASK-006 | Low | API `U10` 与 PRD `U-10` 编号语义冲突 | 否 |
| DOC-001 | Low | 任务引用的 `docs/PRD.md` 实际不存在 | 否 |

总计：**1 Critical、13 High、12 Medium、2 Low**。

---

## 3. 详细问题

### ARC-001：Memory 长期记录写入权与 Go 权威边界矛盾

**严重程度：Critical**

**问题描述**

- `TECH_DESIGN` §10.10 使用“Go/Python Memory Service”共同校验并写入 Memory Metadata，职责不清。
- `DATABASE_DESIGN` §1.3 明确规定 Python 只拥有 `mv_ai`，不得写 `mv_core`。
- `DATABASE_DESIGN` §6.4 将 `npc_memories`、`npc_conversation_summaries`、`npc_memory_snapshots` 放在 `mv_core`。
- `TASKS` `P5-PY-002` 却要求 Python 实现长期 MemoryRepository，并把 `npc_memories/secret_payloads` 列为 Python 数据依赖。
- `TECH_DESIGN` §12 和 `API_DESIGN` 没有提供 Python 写入 Memory 的正式内部接口。

**涉及文档**

`TECH_DESIGN` §1.2、§7.1、§10.3、§10.10、§12；`DATABASE_DESIGN` §1.3、§6.4、§11.4；`TASKS` §0.3、`P5-GO-002`、`P5-PY-002`、`P5-PY-005`。

**产生的风险**

- 实施者若按 `P5-PY-002` 执行，要么给 Python 开放 `mv_core` 写权限，直接违反 Go 权威红线；要么绕过正式契约私自复用数据库账号。
- Python 可能绕过 Go 的可见性、来源事件、关系上限、权限投影和审计校验写记忆。
- `secret_payloads` 若被 Python 直接写入或读取，会破坏 `mv_core` 最小权限、KMS 密钥和 SecretService 边界。
- 记忆写入缺少单一审计链，后续无法判定“模型候选”与“权威记忆元数据”谁拥有最终语义。

**修复建议**

1. 推荐将 `npc_memories` 元数据与来源关系保留在 Go 权威侧。
2. Python 只生成 `MemoryCandidate`，通过内部 API 提交给 Go。
3. Go 负责来源事件、actor scope、权限、关系上限、去重、`supersedes` 和最终 MemoryRecord 写入。
4. Python 只读取 Go 裁剪后的 Memory View；禁止直接访问 `mv_core.npc_memories` 和 `secret_payloads`。
5. 若确有高吞吐需求，可在 `mv_ai` 建非权威 Memory staging/cache，但不得成为事实来源。
6. 同步修改 `TECH_DESIGN` §10、`DATABASE_DESIGN` §6.4、`TASKS` `P5-*`。

**是否阻塞 MVP**

不阻塞纯剧本杀 MVP 主闭环。阻塞 Phase 2 NPC/RPG 与 GATE-MEMORY。

---

### ARC-002：WebSocket 公共信封在 TECH 与 API 之间不一致

**严重程度：High**

**问题描述**

- `TECH_DESIGN` §11.3 使用 `messageId`、`stream`、`streamId`、`seq`、`type`、`serverTime`、`projectionVersion`。
- `API_DESIGN` §8.2 使用 `event_id`、`event_type`、`room_id`、`session_id`、`sequence`、`timestamp`、`payload`。
- API 将 `stream` 与 `schema_version` 放在“扩展字段”中，不是必填字段，但又规定“同一 stream 的 sequence 单调递增”。缺少必填 `stream` 时，`sequence` 无法确定比较空间。
- API 使用 `session_id` 表示对局 ID，其他 HTTP 契约和 UI 均使用 `game_id`。
- `TECH_DESIGN` §11.1 写 `POST /realtime/ticket`，`API_DESIGN` R11 写 `POST /api/v1/realtime/tickets`。
- TECH 要求按 `messageId + seq` 去重，API 要求按 `event_id + sequence` 去重，没有正式映射。

**涉及文档**

`TECH_DESIGN` §11.1、§11.3、§11.6、§11.7；`API_DESIGN` §8.1、§8.2、§8.8、§8.9；`UI_DESIGN` §2.5、§8.6。

**产生的风险**

- Go、Python/前端生成类型、WebSocket reducer 和测试 fixture 会产生字段漂移。
- 房间事件与游戏事件无法稳定判断序号空间，重连补差和缺口检测可能误判。
- `session_id` 与 `game_id` 双命名会造成客户端模型、日志字段和数据库字段映射错误。
- 协议版本和载荷 Schema 非必填，会使旧客户端与新投影在无提示情况下混用。

**修复建议**

1. 只保留一份规范 WebSocket Envelope，建议在 `packages/contracts/json-schema/ws-messages/` 冻结。
2. 必填 `protocol_version`、`schema_version`、`stream`、`stream_id`、`seq`、`event_id`。
3. 明确 `event_id` 是否等同 `messageId`，不要在文档中并存两个主去重键。
4. 对局统一使用 `game_id`；若协议层坚持 `session_id`，必须定义等价映射并禁止双写。
5. 统一 ticket endpoint 为 API 定义的 `/api/v1/realtime/tickets`。

**是否阻塞 MVP**

是。GATE-WS 通过前不得开始房间/游戏实时联调。

---

### ARC-003：Go/Python 内部 AI 协议缺少字段级完整契约和回调防环规则

**严重程度：High**

**问题描述**

- `TECH_DESIGN` §12 列出了内部 endpoint 名称和原则，但没有完整 request/response schema、错误码矩阵、取消与恢复状态机、流式中断语义和 callback ACK 规则。
- `API_DESIGN` 明确说明内部 AI API 不在公共 API 文档内，且当前仓库没有内部 OpenAPI/JSON Schema 文件。
- `TASKS` `P2-DB-002` 被设计为后续补契约，但多个 P0 Go/Python 任务以“内部 AI API 全集”为依赖。
- `TECH_DESIGN` 的调用图同时存在 Go → Python 和 Python → Go callback/context resolve。没有明确禁止同步回合中 Python 再次回调 Go 并形成嵌套调用环。

**涉及文档**

`TECH_DESIGN` §3.1、§5.7、§7.7、§10.10、§12.2-§12.9；`API_DESIGN` §12.4；`TASKS` `P2-DB-002`、`P2-GO-002`、`P4-PY-001`、`P5-GO-001`。

**产生的风险**

- Go 与 Python 对 timeout、retry、idempotency、partial failure 和取消的理解可能不同。
- 回调无 Ack/Correlation 规则时，Python 可能重复注册 artifact、重复提交 intent 或重复写 progress。
- `Go → Python DM/NPC → Python 回调 Go Context Resolve → Go 再次调用 AI` 的运行时环可能导致死锁、请求放大和 trace 断裂。

**修复建议**

1. 在 `P2-DB-002` 前先产出正式 `internal-ai.yaml` 和 JSON Schema，作为唯一契约源。
2. 明确定义同步回合输入只能由 Go 在调用前组装；Python 在该同步回合内不得再次请求 Go context。
3. 回调接口必须包含 `run_id + request_id + trace_id + job_id + schema_version + idempotency_key` 和 ACK 状态。
4. 定义传输错误、Schema 错误、安全阻断、超时、取消、人工等待和降级响应。
5. 在 trace 中记录调用方向与调用深度，并设置最大回调深度。

**是否阻塞 MVP**

是。内部 AI 契约未冻结前，不得开始 Go/Python 集成和 DM 联调。

---

### ARC-004：上下文编译职责在 Go 与 Python 间重叠

**严重程度：Medium**

**问题描述**

- `TECH_DESIGN` §5.7、§6.9、§18.4 将授权知识切片和 `ContextProjector` 放在 Go。
- `TECH_DESIGN` §3.2、§7.1 又描述 Python AI API 承担“上下文编译”。
- 没有明确“权限裁剪”和“Python 内部排序、摘要、token 组装”的边界。

**涉及文档**

`TECH_DESIGN` §3.2、§5.7、§6.9、§7.1、§10.9、§18.4；`TASKS` `P3-GO-005`、`P5-PY-004`。

**产生的风险**

- 两处都可能实现对象检索或权限判断，形成重复业务逻辑。
- 一旦 Python 自行补查询或缓存全量实体，ACL 会从“检索前强过滤”退化为“模型侧自律”。
- 两个团队可能对 token 预算裁剪和上下文 Manifest 责任产生分歧。

**修复建议**

1. 明确 Go 负责授权、对象级/字段级裁剪、revealGate 和允许实体集合。
2. Python 只负责对 Go 返回的 `AllowedKnowledgeSlice` 做确定性排序、摘要、压缩和 token 预算组装。
3. Python 不得通过实体 ID 补查 Go 数据。
4. `contextManifest` 应由 Go 和 Python 共同签名：Go 记录授权实体，Python 记录最终注入顺序和预算。

**是否阻塞 MVP**

否，但应在 AI DM 上线前收敛。

---

### ARC-005：`Project` / `Script` 规范命名尚未在 TECH 中收敛

**严重程度：Medium**

**问题描述**

- PRD 使用 `Project` / `ProjectVersion`。
- `DATABASE_DESIGN` §0.2、§0.3 已明确统一为 `scripts` / `script_versions`。
- `TASKS` GAP-06 明确采用数据库命名。
- `TECH_DESIGN` 仍大量使用 `projects`、`project_versions`、`ProjectVersion`、`projectId`，其 §13.1、§15、§14.1 仍未同步。

**涉及文档**

`MysteryVerse-PRD.md` §15；`TECH_DESIGN` §5.2、§9.2、§13.1、§14.1、§15；`DATABASE_DESIGN` §0.2-§0.3；`TASKS` GAP-06。

**产生的风险**

- 跨语言字段可能继续生成 `project_id`，而数据库只有 `script_id`。
- 对象存储路径、日志字段、AI State 和任务参数会出现双命名。
- 迁移脚本和 API adapter 需要额外映射层，增加长期维护成本。

**修复建议**

1. 以 `script` 为 API 与数据库正式资源名。
2. `project` 只允许作为产品/创作语境中的同义词，不得进入代码 DTO、数据库列和协议字段。
3. 更新 TECH 中所有 `projectId` 为 `scriptId`，或明确声明“文档内部概念映射”。

**是否阻塞 MVP**

否，但必须在代码生成和迁移开始前收敛。

---

### LNG-001：LangGraph 拓扑、TASKS 节点和 UI 节点映射不一致

**严重程度：High**

**问题描述**

- `TECH_DESIGN` §8.2 的节点清单没有独立 `TaskNode`，但 `ClueNode` 输出中包含“任务”。
- `TASKS` `P2-PY-008` 定义了独立 `TaskNode`，并让 `PlotNode` 依赖 `ClueNode + TaskNode`。
- `TECH_DESIGN` §8.3 流程图没有 `TaskNode` 节点，也没有从 `TaskNode` 汇入 Validation 的边。
- `TASKS` `P2-PY-012` 与 `P2-PY-013` 将确定性 Validation 和语义 Validation 拆成两个节点，但 TECH 图只有一个 `ValidationNode`。
- `UI_DESIGN` §4.15 的节点展示映射同样没有 `TaskNode` 和独立语义校验节点。

**涉及文档**

`TECH_DESIGN` §8.2、§8.3、§8.4、§15.2；`TASKS` `P2-PY-007`、`P2-PY-008`、`P2-PY-012`、`P2-PY-013`、`P2-PY-017`；`UI_DESIGN` §4.15、§5.6。

**产生的风险**

- 按 TASKS 实现后，`TaskNode` 可能没有注册、没有边、没有 checkpoint 汇合条件，导致角色目标/任务生成被跳过。
- 确定性校验与语义校验可能被错误合并或只执行一部分。
- UI 无法展示真实节点状态，进度权重和重试入口可能与后端节点不一致。
- Graph State 中 `quests`、`validationIssues` 的 reducer 契约无法稳定定义。

**修复建议**

1. 冻结唯一 LangGraph 节点表，明确 `ClueNode` 是否负责任务；建议保留独立 `TaskNode`，避免职责过宽。
2. 更新拓扑为 `ClueNode + TaskNode -> PlotNode` 或 `ClueNode -> TaskNode -> PlotNode`，并明确汇合 reducer。
3. 将 `ValidationNode` 定义为子图，包含 `DeterministicValidationNode` 与 `SemanticReviewNode`，或明确两个独立节点及边。
4. 同步更新 UI 节点映射、进度权重、TASKS 依赖和 Golden Fixture。

**是否阻塞 MVP**

是。节点拓扑不稳定会直接阻断 LangGraph 完整集成和生成闭环。

---

### LNG-002：自检无法证明线索可推导、真相唯一和阻断矛盾零漏检

**严重程度：High**

**问题描述**

- PRD 要求“案件真相可从线索推导”“阻断级一致性问题不得发布”“不可推导结局漏检率为 0”。
- TECH 的 Validation 包含 Schema、引用、可达性、规则模拟和模型评审，但没有定义从“线索集合”到“唯一结论”的形式化证明方法。
- 单纯阶段图可达性和线索引用完整性不能证明：
  - 玩家拥有足够信息得出凶手；
  - 其他候选解释被有效排除；
  - 多条线索之间没有互相覆盖或循环证明；
  - 某条关键线索缺失时案件是否仍可解；
  - 语义评审模型是否漏检。
- 模型评审是概率性手段，不能承担“阻断漏检率为 0”的发布门禁。

**涉及文档**

`MysteryVerse-PRD.md` §9.3.8、§11.11、§19.4；`TECH_DESIGN` §8.6、§9.7；`TASKS` `P2-PY-012`、`P2-PY-013`、`P8-TEST-PY-004`。

**产生的风险**

- 系统可能生成“引用完整但推不出”的剧本并允许发布。
- 校验结果无法解释，作者只能依赖人工试玩。
- P8 发布的“阻断漏检率 0”无法形成可复现证明。

**修复建议**

1. 增加正式 `DerivationPlan` 和 `TruthWitness`：
   - 每个真相事实必须由至少一组可获取线索和推理步骤支撑；
   - 每条关键线索记录它排除哪些替代解释；
   - 发布前生成可机器检查的推导路径。
2. 对 MVP 可采用约束求解或有限状态穷举模拟，验证：
   - 至少一条有效解；
   - 不存在满足全部线索但结论冲突的解；
   - 删除任一必需线索后解不成立，或标记为冗余。
3. 语义评审只作为警告来源，不承担阻断级零漏检证明。
4. 将推导证明结果纳入 ValidationReport 和发布门禁。

**是否阻塞 MVP**

是。剧本杀的核心可信度就是“线索可推导”，不能仅靠模型评审。

---

### LNG-003：内容生成总超时与阶段超时冲突

**严重程度：Medium**

**问题描述**

- PRD §11.5 规定内容生成总超时 120 秒。
- PRD §11.9 又规定内容生成阶段超时为 300 秒，并允许一次重试。
- TECH 的异步工作流没有明确父级 Run deadline 与单节点 deadline 的父子预算关系。

**涉及文档**

`MysteryVerse-PRD.md` §11.5、§11.9；`TECH_DESIGN` §8.7、§12.1、§16.5；`TASKS` `P2-PY-016`。

**产生的风险**

- 单节点 300 秒超时不可能落在 120 秒总超时内。
- 重试预算可能超过父任务 deadline，导致 Worker 已判定失败但节点仍继续执行。
- 成本和用户等待时间不可预测。

**修复建议**

1. 区分：
   - Agent 单次执行总超时；
   - Workflow 总执行预算；
   - 单节点超时；
   - Provider 单请求超时；
   - 重试退避预算。
2. 父级 deadline 必须向子节点传播，子节点不得获得超过剩余预算的 deadline。
3. 建议将“120 秒”改为短实时生成档，“300 秒”用于长文档异步阶段，并在 TECH 中写明适用任务类型。

**是否阻塞 MVP**

否，但必须在实现超时框架前修正。

---

### MEM-001：PRD `PlayerContext` 未在 TECH、DB、TASKS 中落地

**严重程度：Medium**

**问题描述**

- PRD §11.4.8 定义玩家上下文管理 Agent，§15.6 定义 `PlayerContext` 实体。
- TECH 只在内部 `actorType` 中列出 `player_context`，没有定义其生命周期、schema、持久化、恢复或与 Memory 的关系。
- DATABASE_DESIGN 没有 `player_contexts` 表或对应加密载荷约定。
- TASKS 没有独立任务实现玩家上下文管理。

**涉及文档**

`MysteryVerse-PRD.md` §11.3、§11.4.8、§11.6.3、§15.6；`TECH_DESIGN` §10、§12.5；`DATABASE_DESIGN` §3、§6；`TASKS` Phase 2、Phase 5。

**产生的风险**

- AI DM 多轮问答或跨阶段摘要可能依赖临时拼接，重连后上下文不一致。
- 可能把某玩家上下文误用于其他玩家或 AI NPC。
- PRD 中的 Agent 能力和数据实体没有验收载体。

**修复建议**

1. 明确 MVP 是否采用“每次从可见投影重建”的无持久 PlayerContext 方案。
2. 若采用持久摘要，应定义 `PlayerContextSnapshot` 作用域、版本、过期、来源事件和加密策略。
3. 增加到 Context Resolve、Memory Snapshot 或独立玩家上下文任务。

**是否阻塞 MVP**

否，若 MVP 限定为每次投影重建。若要求跨阶段玩家上下文摘要，则升级为 MVP 阻塞。

---

### MEM-002：长期记忆缺少版本字段与并发约束

**严重程度：Medium**

**问题描述**

- TECH §10.5 的 `MemoryRecord` 包含 `version`。
- DATABASE_DESIGN §6.4.3 的 `npc_memories` 只有 `supersedes_memory_id`、`dedupe_key` 和生命周期字段，没有 `version_no` / `record_version`。
- TASKS `P5-PY-002` 要求“版本化更新”和“新记忆替代旧记忆”，但没有并发版本条件。

**涉及文档**

`TECH_DESIGN` §10.5、§10.10；`DATABASE_DESIGN` §6.4.3、§14.3；`TASKS` `P5-DB-001`、`P5-PY-002`、`P5-PY-005`。

**产生的风险**

- 两个 Worker 同时生成同一 NPC 记忆时可能互相覆盖。
- `supersedes` 链可能出现分叉，无法判断哪一版本是当前有效记忆。
- 快照恢复后无法用版本条件判断快照是否落后。

**修复建议**

1. 为 `npc_memories` 增加 `version_no` 或 `record_version`。
2. 采用追加式版本，不原地覆盖摘要。
3. 增加 `current_memory_key` 或唯一有效版本约束，保证同一逻辑记忆只有一个当前有效版本。
4. 写入使用 `expected_version` 或来源事件序号做乐观并发。

**是否阻塞 MVP**

否，阻塞 Phase 2 Memory/NPC。

---

### ENG-001：聊天、DM 等事件与 `state_version` / `event_seq` 语义冲突

**严重程度：High**

**问题描述**

- DATABASE_DESIGN §12.2 规定 `game_sessions.state_version` 每次成功命令或系统推进严格加 1。
- 同一文档要求 `game_states.last_event_seq` 与 `game_sessions.current_event_seq` 在同一事务更新并保持一致。
- API G05 发送聊天后返回 `state_version: 18`，但产生新的 `sequence: 186`。
- API G08 AI DM 也返回原有 `state_version`，同时可能持久化 AI 消息事件。
- 文档没有区分：
  - 状态变更命令版本；
  - 纯事件/叙事版本；
  - 投影版本；
  - 消息序号；
  - 游戏事件序号。

**涉及文档**

`DATABASE_DESIGN` §6.3.4、§6.3.7、§6.3.8、§12.2、§12.3、§12.10；`API_DESIGN` G05、G08、§8.2；`TECH_DESIGN` §6.4、§11.3。

**产生的风险**

- “成功命令必增 state_version”与聊天/DM 返回示例互相矛盾。
- `last_event_seq` 不变量可能被消息事件破坏。
- 重连、快照恢复和规则回放无法确定某事件是否应改变状态。
- 客户端可能用错误版本重试，造成无谓冲突或幂等误判。

**修复建议**

1. 显式定义三种版本：
   - `state_version`：仅状态机/规则状态变更增加；
   - `event_seq`：所有权威事件单调增加；
   - `projection_version`：投影 Schema 或裁剪结果版本。
2. 明确聊天、DM、系统公告是否属于权威 GameEvent。
3. 若属于 GameEvent，更新 `current_event_seq` 并维护可校验的 `last_event_seq`，但不强制增加 `state_version`。
4. 修改 DATABASE_DESIGN 的“每次成功命令”表述，并更新 API 示例与测试不变量。

**是否阻塞 MVP**

是。该语义不稳定会直接影响实时、重连和回放。

---

### ENG-002：局内暂停/恢复只依赖房间序号的契约不足

**严重程度：High**

**问题描述**

- API R10 将 `pause` / `resume` 放在房间 `host-actions` 下，只要求 `expected_room_event_seq`。
- TECH/DATABASE 规定局内状态、计时器和状态机由 GameSession 行锁、owner lease 和 `expected_state_version` 保护。
- 局内暂停会改变阶段计时和游戏状态，但 API 请求没有 `game_id`、`expected_version` 或游戏幂等键。

**涉及文档**

`API_DESIGN` R10、§10.5；`TECH_DESIGN` §6.5、§6.6、§6.13、§11.9；`DATABASE_DESIGN` §6.3.15、§12.5、§12.11；`MysteryVerse-PRD.md` R-08。

**产生的风险**

- 暂停请求可能绕过 Runtime owner 的串行化路径。
- 同一时刻阶段推进与暂停请求可能基于不同权威版本。
- room event seq 成功而 game state/timer 更新失败时，状态不一致。
- 审计无法判断暂停命令对应哪个游戏状态版本。

**修复建议**

1. 将局内暂停/恢复定义为游戏控制命令，而非普通房主房间操作。
2. 单独提供 `POST /games/{game_id}/control` 或在 R10 中强制加入 `game_id`、`expected_version` 和 `Idempotency-Key`。
3. 在同一 GameSession 事务中更新 timer、state 和游戏事件，再通过 Outbox 写 room event。
4. 房主身份只作为授权条件，不作为绕过 Runtime 的路径。

**是否阻塞 MVP**

是。PRD 将暂停/恢复列为 P0 房间能力。

---

### DATA-001：邀请 token 表缺失

**严重程度：High**

**问题描述**

- API R07/R08 和 PRD 需要可创建、解析、撤销、限制次数的房间邀请 token。
- TASKS `P1-GO-005` 明确要求读写 `invitation_tokens`。
- DATABASE_DESIGN 表清单、字段定义、索引和 DDL 都没有 `invitation_tokens`。
- `share_links` 不能替代房间邀请 token，因为其目标、权限、使用次数和解析流程不同。

**涉及文档**

`MysteryVerse-PRD.md` §9.4、§14.9；`API_DESIGN` R04、R07、R08；`DATABASE_DESIGN` §3、§6、§7、§16；`TASKS` `P1-GO-005`。

**产生的风险**

- 邀请链接只能靠临时 JSON、`share_links` 或 room code 近似实现，无法保证过期、撤销、使用上限和审计。
- 加入房间权限可能绕过邀请范围。
- 数据库无唯一约束，重复使用和并发兑换难以测试。

**修复建议**

1. 增加 `invitation_tokens` 表，至少包含：
   - `room_id`、`created_by`
   - `token_hash`
   - `max_uses`、`use_count`
   - `expires_at`、`revoked_at`
   - `invitee_user_ids_json` 或关联表
   - `created_at`
2. 建立 `UNIQUE(token_hash)`、`INDEX(room_id, expires_at)` 和并发使用约束。
3. 邀请兑换日志或 room event 需记录脱敏 token 引用。
4. 同步更新表清单、字段定义、索引、DDL、权限与 TASKS。

**是否阻塞 MVP**

是。房间邀请是核心开局流程。

---

### DATA-002：26 字符 ULID 与带前缀示例的长度口径冲突

**严重程度：Medium**

**问题描述**

- DATABASE_DESIGN 将所有 ID 定义为 `CHAR(26)`。
- API_DESIGN 明确写“ID 为 26 字符 ULID 字符串”，但示例大量使用 `script_01J...`、`game_01J...`、`room_01J...`、`evt_...`、`msg_...`。
- TECH 示例同样使用 `project_...`、`game_...`、`pkg_...` 等前缀 ID。

**涉及文档**

`DATABASE_DESIGN` §1.2、§5.1；`API_DESIGN` §1.1、§1.5 及各接口示例；`TECH_DESIGN` §6.4、§9.2、§11.3。

**产生的风险**

- 生成模型可能选择带前缀字符串，数据库 `CHAR(26)` 无法存储。
- Schema 的正则、长度和主键约束可能互相冲突。
- 前端、日志和 trace 的 ID 解析与分组逻辑不稳定。

**修复建议**

1. 明确二选一：
   - 纯 26 字符 ULID，前缀只存在于展示层；
   - 变长带类型前缀 ID，例如 `CHAR(32)` 或 `VARCHAR(40)`。
2. 若选择纯 ULID，删除协议示例中的语义前缀。
3. 在 OpenAPI/JSON Schema 中统一 `pattern` 和 `minLength/maxLength`。

**是否阻塞 MVP**

是，在契约冻结前。ID 格式是跨数据库、API、WS、日志和生成代码的基础约束。

---

### DATA-003：大厅搜索、筛选和评分读模型缺索引或聚合结构

**严重程度：Medium**

**问题描述**

- API S02 支持 `q`、人数范围、时长上限、难度、标签、评分和热门排序。
- DATABASE_DESIGN 的主要索引只覆盖 `scripts(game_type,status,published_version_id)`、标题普通索引和标签关系。
- 没有为 `min_players/max_players/duration_minutes/difficulty` 组合筛选建立明确索引。
- API S02/S03 返回聚合评分 `rating` / `score + count`，但没有 `script_statistics`、评分汇总表，也没有说明如何高效聚合 `script_reviews`。
- 中文标题/简介搜索没有定义 MySQL FULLTEXT ngram、搜索引擎或前缀搜索策略。

**涉及文档**

`API_DESIGN` S02、S03、C01；`DATABASE_DESIGN` §6.2.5、§6.6.2、§7.3；`TASKS` `P1-GO-004`。

**产生的风险**

- 数据量增长后大厅筛选和搜索退化为全表扫描。
- 评分列表查询每次聚合评论表，导致公开详情延迟上升。
- “热门”排序口径不稳定，无法复现。

**修复建议**

1. 增加 `script_statistics` 或等价聚合表，保存评分、评分人数、完成局数、最近游玩和更新时间。
2. 为筛选条件建立组合索引或明确读副本/搜索服务方案。
3. 中文全文搜索应明确使用 MySQL ngram FULLTEXT 或后续独立搜索索引。
4. 将聚合刷新写入 Outbox/Worker 任务。

**是否阻塞 MVP**

小规模 MVP 可通过读副本或内存聚合临时支持，但公开规模前必须修复。

---

### API-001：运营审核后台缺少 API、任务和 UI 接入覆盖

**严重程度：High**

**问题描述**

- PRD Phase 1 包含内容审核、举报基础、后台下架、事故排查和最小审计。
- UI_DESIGN 定义了 `/admin/content`、`/admin/reports`、`/admin/generation`、`/admin/incidents`。
- API_DESIGN 明确“不含后台审核 API 的完整运营权限流程”。
- TASKS 没有独立的后台 API、后台前端或完整审核工单任务。`P2-GO-007` 只覆盖发布校验/下架相关内部逻辑，`P8-INF-001` 主要覆盖观测看板。

**涉及文档**

`MysteryVerse-PRD.md` §9.11、§17.1；`UI_DESIGN` §5.19、§9.8；`API_DESIGN` §12.4；`TASKS` GAP-05、`P2-GO-007`、`P8-INF-001`。

**产生的风险**

- 内容可以发布，但运营无法完成审核、举报处理、敏感访问审批和事故干预。
- 公开 MVP 不满足 PRD 的上线合规与运营门槛。
- 敏感后台操作缺少正式权限、审批、审计和 UI 任务，可能临时复用玩家 API。

**修复建议**

1. 新建独立 `ADMIN_API` 契约，不复用玩家公共 API。
2. 增加后台认证、RBAC + ABAC、审批、双人复核和审计任务。
3. 增加 `ADMIN-FE-*` 任务覆盖内容、举报、生成任务和事故页面。
4. 将“公开内容审核、下架、举报处理、事故冻结”明确为公开 MVP 门禁。

**是否阻塞 MVP**

是，阻塞公开 MVP。私有试玩闭环可先不依赖完整后台。

---

### API-002：编辑器候选 Diff、Artifact 查询和接受/拒绝接口缺失

**严重程度：High**

**问题描述**

- PRD 要求局部重生成后查看 Diff、逐项接受/拒绝。
- UI §5.7 要求 `Diff Panel`、图片候选接受/拒绝、来源对象和 artifact 状态。
- TASKS `P2-FE-005` 依赖“artifact 查询”，但没有对应公共 API。
- TASKS `P2-FE-008` 要求候选 job 状态、Diff 选择、接受/拒绝，但 API S16 只返回 `job_id + candidate_id`。
- API_DESIGN 没有：
  - 获取 regeneration candidate / Diff；
  - 接受或拒绝候选；
  - 查询 artifact 元数据；
  - 获取图片候选与审核状态。

**涉及文档**

`MysteryVerse-PRD.md` §9.3.9、§14.7；`UI_DESIGN` §4.15、§5.7；`API_DESIGN` S13-S17；`TASKS` `P2-GO-006`、`P2-FE-005`、`P2-FE-008`。

**产生的风险**

- 编辑器核心“AI 候选人工审核”流程无法通过正式 API 完成。
- 前端可能直接读取对象存储候选或内部 artifact 接口，绕过权限与契约。
- 人工接受/拒绝无法形成可审计、幂等的状态转换。

**修复建议**

增加正式公共接口，例如：

- `GET /api/v1/scripts/{script_id}/versions/{version_id}/candidates/{candidate_id}`
- `POST /api/v1/scripts/{script_id}/versions/{version_id}/candidates/{candidate_id}/actions`
- `GET /api/v1/scripts/{script_id}/versions/{version_id}/artifacts`
- `GET /api/v1/artifacts/{artifact_id}`，仅 owner/reviewer 可见脱敏元数据

候选操作必须携带 `expected_row_version`、`Idempotency-Key`、锁定字段和审计理由。

**是否阻塞 MVP**

是。AI 剧本工坊的核心差异化和人工修订闭环依赖该能力。

---

### API-003：等待房聊天仍无 API，但 PRD/UI 将其作为页面能力

**严重程度：High**

**问题描述**

- PRD §14.9 的等待房核心组件包含等待聊天。
- UI §5.9 将聊天与规则摘要作为等待页侧栏，并在 §9.8 明确列为契约缺口。
- API_DESIGN 只有游戏内 `G04/G05`，没有等待房消息 REST/WS 契约。
- TASKS GAP-04 采用“显示未接入”的临时策略，但 PRD 没有正式移出该需求。

**涉及文档**

`MysteryVerse-PRD.md` §14.9；`UI_DESIGN` §5.9、§9.8；`API_DESIGN` §5、§8；`TASKS` GAP-04、`P1-DB-007`、`P4-FE-001`。

**产生的风险**

- 等待页出现已完成视觉设计但不可用的核心交互。
- 实施者可能错误复用游戏内消息接口，扩大游戏权限到等待房。
- 如果不实现，必须经过产品范围变更，而不是在任务里静默降级。

**修复建议**

1. 增加等待房消息契约，建议使用 `room_id + room_seq`，与游戏消息分开。
2. 定义可发言成员、私聊是否允许、审核、限流、幂等、保留周期和 WS 事件。
3. 若 MVP 暂不做，必须在 PRD 变更记录中显式标为延期，并保留需求，不直接删除。

**是否阻塞 MVP**

是，按当前 PRD 页面能力。若产品正式批准延期，可降为非阻塞。

---

### API-004：完整历史、通知、成就和首页聚合的 DB/API 契约不完整

**严重程度：Medium**

**问题描述**

- PRD 定义完整游戏历史、通知、成就和首页推荐/精选。
- UI 明确将这些列为契约缺口。
- API 只有 `U04.include=recent_games`，没有完整 `GET /me/games`。
- DATABASE_DESIGN 没有通知表和成就表。
- TASKS `P1-DB-006`、`P1-DB-007` 标为 BLOCKED。

**涉及文档**

`MysteryVerse-PRD.md` §9.1、§9.2、§14.1、§14.16-§14.17、§15.3；`DATABASE_DESIGN` §3、§6；`API_DESIGN` U04；`UI_DESIGN` §9.8；`TASKS` GAP-01、GAP-03、GAP-05。

**产生的风险**

- 页面只能显示禁用或空状态，Phase 1 页面完整度不足。
- 通知和成就若临时用统计字段模拟，会产生不可维护的伪实体。
- 完整历史检索可能临时复用近期列表，导致分页、筛选和权限不可控。

**修复建议**

1. 将通知、成就、完整历史和首页聚合分别拆成小契约任务，不再由一个宽范围 BLOCKED 任务承载。
2. 增加最小表与 API，或明确正式 feature flag 和 PRD 范围变更。
3. 首页推荐在 MVP 可继续采用公开搜索兜底，但需定义官方精选配置来源。

**是否阻塞 MVP**

不阻塞纯核心对局，但阻塞 Phase 1 完整页面验收。

---

### API-005：UI 使用 API 未定义的错误码语义

**严重程度：Medium**

**问题描述**

- UI §8.6 和 §9.7 把 `IDEMPOTENT_REPLAY` 当作错误码处理，但 API 将其定义为成功响应中的 `meta.idempotent_replay=true`。
- UI 使用 `REALTIME_DISCONNECTED` 与 `SERVICE_UNAVAILABLE`，API 错误码表没有这两个 Code。
- PRD §14.19 同时列出这些通用建议码，导致 API 与 UI 都引用旧口径。

**涉及文档**

`MysteryVerse-PRD.md` §14.19；`API_DESIGN` §1.3、§1.4、§9；`UI_DESIGN` §8.6、§9.7。

**产生的风险**

- 前端错误分支无法命中后端实际返回值。
- 幂等成功回放可能被当成错误，触发错误 Toast。
- 断线/维护状态依赖前端本地推断，与 API 状态不一致。

**修复建议**

1. 将 `IDEMPOTENT_REPLAY` 从错误码映射中移除，统一读取 `meta.idempotent_replay`。
2. 对 `REALTIME_DISCONNECTED` 和 `SERVICE_UNAVAILABLE` 二选一：
   - 加入 API 统一错误码；
   - 明确定义为客户端连接状态或网关错误。
3. 生成 TS 错误码枚举，禁止手写字符串。

**是否阻塞 MVP**

否，但实现前必须统一。

---

### API-006：客户端消息允许 `system_note`，存在伪造系统消息风险

**严重程度：Medium**

**问题描述**

- PRD 明确“客户端不得自行生成系统消息”。
- API G05 的 `message_type` 允许客户端提交 `system_note`。
- 文档没有说明 `system_note` 是否只是用户备注、系统样式还是系统消息数据。

**涉及文档**

`MysteryVerse-PRD.md` §9.8.3；`API_DESIGN` G05；`UI_DESIGN` §5.10。

**产生的风险**

- 用户可能生成与系统播报相似的 UI 元素，造成社交欺骗或误操作。
- 审核与复盘无法区分服务端系统事件与用户文本。

**修复建议**

1. 移除客户端 `system_note` 类型，或重命名为 `user_note`。
2. 系统消息只能由 Go Runtime、Timer Worker、AI 安全模板或运营审计动作生成。
3. 增加伪造 `message_type=system` 的负向测试。

**是否阻塞 MVP**

否，但聊天实现前必须修正。

---

### TASK-001：MVP E2E 任务依赖 DEFERRED 的 RPG 页面

**严重程度：High**

**问题描述**

- `P8-TEST-FE-004` 是 P0，验收目标是剧本杀 MVP 端到端闭环。
- 其前置依赖包含 `P7-FE-003`，而 `P7-FE-003` 属于 RPG，状态为 `DEFERRED`、优先级 P2。
- 按 TASKS §15 的状态规则，P0 任务不能在前置未完成时标记 DONE。

**涉及文档**

`TASKS` `P8-TEST-FE-004`、`P7-FE-003`、§13、§15。

**产生的风险**

- MVP 的最终 E2E 门禁永远不能合法启动。
- 测试任务会因未实现的 RPG 被拖到 Phase 7 之后，失去 MVP 提前验收价值。
- 团队可能违规忽略前置依赖，削弱任务协议可信度。

**修复建议**

1. 将 E2E 拆成：
   - `P8-E2E-MVP`：只覆盖剧本杀上传到复盘。
   - `P8-E2E-RPG`：Phase 7 完成后执行。
2. 删除 `P8-TEST-FE-004` 对 `P7-FE-003` 的硬依赖。

**是否阻塞 MVP**

是。MVP 自动化闭环验收无法按当前依赖关系完成。

---

### TASK-002：BLOCKED 的宽范围契约任务阻塞多个 P0 页面任务

**严重程度：High**

**问题描述**

- `P1-DB-006` 同时处理项目列表、生成任务列表和完整游戏历史，状态为 BLOCKED。
- `P1-FE-003`（首页/大厅 P0）、`P2-FE-001`（工坊 P0）、`P4-FE-007`（结算/复盘/历史 P0）都把 `P1-DB-006/GAP-*` 作为前置。
- GAP-01 已有“使用 S02 + R02 兜底”的默认处置，本不需要等待完整聚合契约。
- GAP-02 也允许单项目运行，GAP-03 只影响完整历史，不应阻塞结算和复盘。

**涉及文档**

`TASKS` GAP-01、GAP-02、GAP-03、`P1-DB-006`、`P1-FE-003`、`P2-FE-001`、`P4-FE-007`。

**产生的风险**

- 一个 BLOCKED 宽任务锁住多个 MVP 页面，导致前后端无法并行。
- 不同缺口需要不同决策，却被捆绑到同一前置。
- 页面任务可能被迫等待非核心历史/项目聚合契约。

**修复建议**

1. 拆分 `P1-DB-006` 为：
   - 首页聚合兜底决策；
   - 我的剧本/生成任务列表契约；
   - 完整历史契约。
2. 将 `P4-FE-007` 拆为结算、复盘、历史三个任务。
3. 允许结算/复盘按 G06/G07 独立推进，不被完整历史阻塞。

**是否阻塞 MVP**

是。当前依赖图会阻断首页、工坊和结算页面。

---

### TASK-003：可选 P1 功能反向硬阻塞 P0 任务

**严重程度：High**

**问题描述**

- TECH 明确图片生成流程默认关闭或受配额。
- `P2-PY-011` 为 P1，但在可并行任务和依赖图中仍被 `P2-PY-017` 这个 P0 完整工作流集成任务硬依赖。
- `P4-FE-007` 为 P0，却依赖 `P4-PY-004` 的 P1 AI 复盘点评。
- MVP 任务清单又包含全部 `P2-PY-001` 至 `P2-PY-017`，模糊了可选与必做边界。

**涉及文档**

`TECH_DESIGN` §8.2、§8.3、§20.1；`TASKS` `P2-PY-010`、`P2-PY-011`、`P2-PY-017`、`P4-PY-004`、`P4-FE-007`、§13.1。

**产生的风险**

- 可选图片能力会拖慢 MVP 文本生成链路。
- AI 点评不可用时，结算/复盘页面任务被错误视为未完成。
- P0/P1 优先级失去规划意义。

**修复建议**

1. 将 `P2-PY-011` 正式标为 DEFERRED 或 P1 optional，`P2-PY-017` 允许 Image branch disabled 完成。
2. `P4-FE-007` 只依赖确定性结果和结构化复盘，AI 点评作为异步增强任务。
3. 在 Graph 和 UI 中明确 optional branch 的占位与 feature flag。

**是否阻塞 MVP**

是。当前依赖关系会让 P0 任务等待 P1 能力。

---

### TASK-004：多个任务粒度过大，且 P0 TODO 数量失控

**严重程度：Medium**

**问题描述**

- TASKS 当前有 100 个 P0 TODO。
- 多个任务不适合“一次只领取一个任务”的 Agent 协议：
  - `P1-DB-003` 同时创建剧本、生成任务、故事圣经、ACL、角色、场景、阶段、线索、任务、事件等大量迁移。
  - `P1-DB-004` 同时创建房间、游戏、状态、事件、投票、消息、计时器、快照、结算、Outbox。
  - `P2-PY-017` 组装全 Graph 并覆盖全部 fixture。
  - `P4-GO-004` 同时实现阶段控制器、讨论、质询、私聊和 message。
  - `P8-TEST-FE-002` 一次覆盖所有页面状态矩阵。
  - `P8-TEST-X-005` 包含 72 小时稳定性与多类故障注入。

**涉及文档**

`TASKS` §0.3、`P1-DB-003`、`P1-DB-004`、`P2-PY-017`、`P4-GO-004`、`P8-TEST-FE-002`、`P8-TEST-X-005`。

**产生的风险**

- 单个任务变更面过大，难以评审、回滚和记录测试证据。
- 多个团队无法在同一任务内部并行。
- 失败的子能力可能拖垮整个任务状态。

**修复建议**

1. 以“单一聚合/单一页面/单一可验证不变量”为拆分单位。
2. 对数据库任务按领域表组拆分，但允许一个迁移版本包含相关表。
3. 对 Graph 集成拆为拓扑、runtime adapter、fixture、optional branch。
4. 对页面测试矩阵按页面族拆分。
5. 重新区分“MVP P0”和“完整路线 P0”，避免 100 个 P0 并行争抢资源。

**是否阻塞 MVP**

否，但会显著增加延期和执行失控风险。

---

### TASK-005：P5 Memory 被标为 P0，但未纳入 MVP 必做任务集合

**严重程度：Medium**

**问题描述**

- `P5-DB-001`、`P5-GO-001`、`P5-GO-002` 标为 P0 TODO。
- TASKS §13.1 的 MVP 必做集合没有 Phase 5 Memory 任务。
- TECH 将基础 Memory 描述为“Phase 1 后段或 Phase 2 前”。
- PRD Phase 1 明确不进入 RPG/NPC 对话。

**涉及文档**

`TECH_DESIGN` §20.1；`MysteryVerse-PRD.md` §17.1；`TASKS` Phase 5、§13.1。

**产生的风险**

- 执行者无法判断 Memory 是否属于 MVP P0。
- 资源可能过早投入 NPC 记忆，挤压剧本杀闭环。
- P0 标记与 MVP 清单不一致，影响排期和验收。

**修复建议**

1. 若 Memory 不进入剧本杀 MVP，将 Phase 5 整体调整为 P1/DEFERRED。
2. 若至少需要 PlayerContext/短期记忆，将对应子集明确加入 MVP 清单。
3. 保持 PRD 需求不变，只调整阶段与优先级。

**是否阻塞 MVP**

否，但必须先完成范围签署。

---

### TASK-006：API `U10` 与 PRD `U-10` 编号语义冲突

**严重程度：Low**

**问题描述**

- PRD `U-10` 表示“成就与统计”。
- API `U10` 表示“拉黑用户”。
- TASKS `P1-FE-005` 引用 `U04-U07、U10`，容易让执行者误以为成就已有 API。

**涉及文档**

`MysteryVerse-PRD.md` §9.1；`API_DESIGN` §3.1、U10；`TASKS` `P1-FE-005`。

**产生的风险**

- 任务执行者可能误接拉黑接口来实现成就入口。
- 评审追踪需求编号时出现语义错位。

**修复建议**

使用全局稳定编号，例如 `REQ-U-10` 与 `API-U10`，或统一重编号。成就仍应遵循 GAP-05。

**是否阻塞 MVP**

否。

---

### DOC-001：任务引用的 `docs/PRD.md` 实际不存在

**严重程度：Low**

**问题描述**

用户任务要求读取 `docs/PRD.md`，TASKS §0.1 也说明该路径不存在，实际文件为 `docs/MysteryVerse-PRD.md`。

**涉及文档**

`docs/PRD.md`（不存在）、`MysteryVerse-PRD.md`、`TASKS` §0.1。

**产生的风险**

- 工具、脚本或新成员按任务说明读取 PRD 时失败。
- 后续自动检查和文档链接可能指向错误文件。

**修复建议**

二选一：

1. 新增 `docs/PRD.md` 作为稳定入口，仅指向或重命名 PRD；
2. 将所有任务和脚本统一改为 `MysteryVerse-PRD.md`。

**是否阻塞 MVP**

否。

---

## 4. 分项审查结论

### 4.1 Go 与 Python 架构

| 检查项 | 结论 |
| --- | --- |
| Go 和 Python 职责是否明确 | 部分通过；AI 生成/表达边界明确，Memory 与 Context 编译仍冲突 |
| 是否存在重复业务逻辑 | 主流程无严重重复；Context 编译和 Validation 存在职责重叠风险 |
| Python 是否能绕过 Go 修改游戏状态 | 数据库账号设计正确；TASKS P5-PY-002 存在反向实施风险 |
| 服务间通信协议是否完整 | 不完整；内部 AI 缺字段级契约，WebSocket 信封不一致 |
| 超时和重试是否合理 | Provider/节点基本合理；生成总超时与阶段超时冲突 |
| 是否存在循环依赖 | 领域代码无明确循环；运行时 Go/Python callback 环缺少防护 |
| 是否存在不必要微服务拆分 | 未发现；模块化单体和多进程部署合理 |

### 4.2 LangGraph

| 检查项 | 结论 |
| --- | --- |
| State 定义是否完整 | 基本完整，但需同步节点拓扑新增字段和 reducer |
| 节点职责是否单一 | `ClueNode` 与 `TaskNode` 冲突；Validation 子节点未收敛 |
| 并行节点是否存在数据竞争 | 设计已使用只读快照、独立 run 和显式 reducer，原则正确 |
| 条件边是否完整 | 主工作流有失败/修复/人工分支；缺 TaskNode/Semantic Validation 边 |
| 自检是否能发现逻辑错误 | 目前不能证明“可推导”和“零阻断漏检” |
| Repair 是否有最大次数 | 有，默认 3 轮 |
| 图片生成是否与核心解耦 | 有，可选且失败占位；TASKS 依赖需同步调整 |
| Checkpoint 是否支持中断恢复 | 有，Redis + MySQL + Object Storage 分层清楚 |
| 是否支持人工审核 | 有，HumanReviewNode interrupt/resume |
| 生成结果是否有统一 Schema | 目标是统一，但实际 schema 仍需在 P2-DB-001/P2-DB-002 冻结 |

### 4.3 Memory

| 检查项 | 结论 |
| --- | --- |
| 短期和长期记忆是否分离 | 是，进程/Redis 与长期 metadata/object snapshot 分层清楚 |
| 内存与磁盘持久化边界 | 是，本地磁盘非权威 |
| 多实例运行数据丢失风险 | Redis/对象存储/MySQL 路径合理 |
| NPC 记忆跨游戏污染 | 已强制 `game_session_id + game_npc_id` |
| 虚构记忆覆盖真实事实 | 设计以 Go 权威优先，候选记忆有 authorityClass |
| 是否支持快照恢复 | 有 |
| 是否有记忆版本管理 | 概念有，数据库缺版本字段，需补齐 |
| 是否有过期和压缩机制 | 有，TTL、expiresAt、摘要压缩和 supersedes |

### 4.4 游戏引擎

| 检查项 | 结论 |
| --- | --- |
| 三种模式独立规则引擎 | 是，共享运行时与独立 Engine |
| 统一游戏事件模型 | 是，但事件与状态版本语义需修正 |
| 状态快照 | 是 |
| 事件回放 | 是，规则回放与展示回放分离 |
| 并发操作 | 是，owner lease + per-game mutex + MySQL 行锁 + 版本条件 + 唯一约束 |
| 私密信息隔离 | 是，viewer manifest + reveal gate + DTO 投影 |
| 断线恢复 | 是，snapshot + delta + seq |
| LLM 是否直接决定胜负 | 否，规则结果由 Go Engine 裁决 |

### 4.5 数据库与 API

| 检查项 | 结论 |
| --- | --- |
| 所有 API 是否有对应数据结构 | 公共接口有字段级概述；候选 Diff、后台、部分页面聚合仍缺 |
| 是否存在缺索引的高频查询 | 有，公开搜索和大厅筛选/评分读模型不足 |
| 是否存在重复数据 | 主要物化状态合理；需避免 Project/Script 双聚合 |
| Redis 与 MySQL 职责是否冲突 | 未发现，职责清晰 |
| WebSocket 是否有版本和序号 | 有概念，但必填性、字段名与 stream 空间不一致 |
| 是否有幂等处理 | 有，覆盖游戏、房间、任务、消息、AI Run |
| 是否有统一错误码 | 有，但 UI 存在旧码/错误语义 |
| 是否越权读取 | 设计较完整，对象/字段/阶段/频道多层防护 |

### 4.6 UI

| 检查项 | 结论 |
| --- | --- |
| PRD 核心功能是否都有页面承载 | 路由与页面覆盖较完整；部分页面受 API 缺口影响 |
| 是否遵循 Tailwind Design System | 是，Token、语义色、间距、圆角、字体和 motion 都很明确 |
| 三种模式是否有独立视觉体验 | 是，构图、材料、信息焦点不同 |
| AI 工作流是否可视化 | 是，但节点集合需与 LangGraph 一致 |
| 动画是否与业务状态绑定 | 是，明确禁止动画决定业务状态 |
| 是否有加载、失败、断线状态 | 是，状态矩阵完整 |
| 是否有移动端适配 | 是，360/390/768/1024/1440/1920 验收视口明确 |

### 4.7 TASKS

| 检查项 | 结论 |
| --- | --- |
| 是否覆盖所有 MVP 功能 | 剧本杀主闭环覆盖较好；后台、等待聊天和部分页面契约不足 |
| 是否存在无前置依赖的阻塞任务 | 存在，BLOCKED 宽任务阻塞 P0 页面 |
| Go、Python、前端能否并行开发 | 大体可以；当前错误依赖会破坏并行 |
| 每个任务是否有验收标准 | 文档层面覆盖完整 |
| 是否有自动化测试 | 有，测试门禁完整；部分测试任务前置不合理 |
| 是否存在粒度过大的任务 | 有，多个任务远超单任务协议合理范围 |
| 是否存在重复任务 | 未发现明确重复；存在职责重叠导致潜在重复实现 |

---

## 5. 优先修复清单

### 5.1 第一优先级：冻结架构红线与核心契约

1. **ARC-001**：确定 Memory 由 Go 权威写入，Python 只提交 Candidate。
2. **ARC-002**：冻结唯一 WebSocket Envelope、stream、seq、schema version 和 ID 映射。
3. **ARC-003**：完成内部 AI OpenAPI/JSON Schema、错误码、取消/恢复和回调防环。
4. **ENG-001**：定义 `state_version`、`event_seq`、`projection_version` 的正式语义。
5. **ENG-002**：将局内暂停/恢复纳入 GameSession 版本与 owner 控制。
6. **DATA-002**：统一资源 ID 长度与前缀策略。

### 5.2 第二优先级：关闭 MVP 数据与 API 缺口

1. **DATA-001**：增加 `invitation_tokens` 表、索引、DDL 和权限。
2. **API-001**：建立独立后台 API、审核权限和后台前端任务。
3. **API-002**：增加候选 Diff、Artifact 查询和接受/拒绝接口。
4. **API-003**：补充等待房聊天 API，或完成正式产品延期决策。
5. **API-004**：拆分完整历史、通知、成就和首页聚合契约。
6. **DATA-003**：设计评分/内容统计读模型和大厅搜索索引。

### 5.3 第三优先级：收敛 LangGraph 与自检质量

1. **LNG-001**：冻结 TaskNode、Validation 子图、并行汇合和 UI 节点映射。
2. **LNG-002**：增加可机器验证的 DerivationPlan/TruthWitness。
3. **MEM-001**：决定 PlayerContext 的 MVP 形态。
4. **MEM-002**：补齐 MemoryRecord 版本和并发约束。
5. **LNG-003**：统一父任务与节点级超时预算。

### 5.4 第四优先级：修复任务依赖与执行粒度

1. **TASK-001**：拆分剧本杀 MVP E2E 与 RPG E2E。
2. **TASK-002**：拆分 `P1-DB-006`，移除对 P0 页面的宽范围阻塞。
3. **TASK-003**：将图片生成与 AI 点评改为可选异步增强。
4. **TASK-004**：拆分超大数据库、Graph、页面测试和稳定性任务。
5. **TASK-005**：重新定级 Phase 5 Memory 与 MVP 范围。
6. **TASK-006**：统一需求编号。
7. **DOC-001**：统一 PRD 文件路径。

---

## 6. 建议的人工确认项

在修改任何源文档前，需要人工确认以下决策：

1. Memory 元数据是否全部由 Go 写入，Python 是否永远不能直接访问 `mv_core.npc_memories`。
2. WebSocket 主去重键采用 `event_id` 还是 `messageId`，`stream` 是否强制必填。
3. 资源 ID 采用纯 26 字符 ULID，还是允许类型前缀的变长 ID。
4. 聊天、DM、系统公告是否属于权威 `game_events`，是否允许只递增 `event_seq` 而不递增 `state_version`。
5. 局内暂停/恢复是否作为独立游戏控制命令。
6. 等待房聊天是否留在 MVP；如果延期，必须记录产品范围变更。
7. 运营后台是否作为公开 MVP 发布门禁。
8. 玩家上下文是否采用持久快照，还是每次从投影重建。
9. 图片生成是否在 MVP 强制实现，还是默认可关闭。
10. AI 复盘点评是否作为 P0 验收的一部分，还是异步增强。

---

## 7. 未删除任何 PRD 需求声明

本审查没有提出删除 PRD 功能需求。对于等待房聊天、完整历史、通知、成就、后台审核、PlayerContext 和 Memory 等缺口，建议是：

- 补契约；
- 补数据承载；
- 补任务；
- 或由产品明确延期并保留需求；

不建议从 PRD 中静默移除。

---

## 8. 最终审查意见

当前文档体系具备良好的架构骨架和很强的安全、幂等、实时、UI 设计意识，但还不能作为“唯一实施真相”直接冻结。最需要防止的不是功能设计不足，而是以下两类问题：

1. 同一事实在多个文档中采用不同协议、命名或权威归属，导致实现阶段各团队各自选择。
2. 任务图用 DEFERRED/BLOCKED 或可选能力反向阻塞 P0，导致 MVP 验收无法按计划闭环。

建议状态：**Conditional Approval**。

在完成第一、第二优先级的契约与数据修复，并修正任务依赖后，本设计可以进入实施冻结评审。等待人工确认后，再按本报告建议修改相关文档。

---

## 9. 2026-11-21 决策落地附录

### 9.1 状态变化

原审查结论保留为历史记录。Task 1 复核后，10 项决策中仅 ADR-003、ADR-009、
ADR-010 为 Accepted，其余 7 项为 Proposed。待批准 ADR 已有候选收敛文本，
但没有负责人确认，因此不得视为正式架构决策。实现契约、Schema 文件和测试仍未
创建，GATE-API、GATE-WS、GATE-GAMESTATE 仍不通过。

### 9.2 候选收敛记录

| 审查项 | 处置 | 现行权威 |
| --- | --- | --- |
| ARC-001 | 候选收敛，待批准：Go 唯一写 Memory，Python 只提交 Candidate | ADR-001 Proposed |
| ARC-002 | 候选收敛，待批准：统一 `event_id/stream/stream_id/seq/schema_version/game_id` Envelope | ADR-002 Proposed |
| ARC-003 | 决策关闭；字段级内部 OpenAPI 仍需 P2-DB-002 实现 | ADR-003、`docs/api/ai-internal-api.md` |
| ARC-004 | 候选收敛，待批准：Go 授权切片，Python 只排序/摘要/裁剪 | ADR-008 Proposed |
| ARC-005 | 关闭：统一 `script/script_version` | ADR-010 |
| MEM-001 | 候选收敛，待批准：MVP PlayerContext 每次重建，不持久化 | ADR-008 Proposed |
| MEM-002 | 候选收敛，待批准：增加 `record_version`，Go 权威追加式更新 | ADR-001 Proposed |
| ENG-001 | 候选收敛，待批准：event_seq 与 state_version 分离 | ADR-004 Proposed |
| ENG-002 | 候选收敛，待批准：独立 G09/G10，携带 expected_version 与幂等键 | ADR-005 Proposed |
| DATA-001 | 候选收敛，待批准：增加 `invitation_tokens` | `docs/database/schema-design.md` |
| DATA-002 | 关闭：纯 26 字符 ULID，无类型前缀 | ADR-003 |
| API-001 | 候选收敛，待批准：基础后台是公开 MVP 门禁 | ADR-007 Proposed |
| API-003 | 候选收敛，待批准：等待房聊天采用独立 `room_messages/room_chat` | ADR-006 Proposed |
| API-005 | 关闭：`IDEMPOTENT_REPLAY` 改为成功 meta；错误码语义同步 | `API_DESIGN.md` §9 |
| API-006 | 关闭：客户端禁止 `system/system_note` | `API_DESIGN.md` G05 |
| TASK-001 | 关闭：MVP E2E 不再依赖 RPG，新增独立模式 E2E | `TASKS.md` P8 |
| TASK-002 | 部分关闭：首页/工坊/结算不再硬依赖宽 BLOCKED 任务 | `TASKS.md` |
| TASK-003 | 关闭：图片和 AI 点评为可选 P1，不作为 P0 完成前置 | ADR-009 |
| TASK-005 | 关闭：Phase 5 全量 Memory 标为 `DEFERRED` | `TASKS.md` Phase 5 |
| DOC-001 | 关闭：PRD 固定为 `docs/MysteryVerse-PRD.md` | ADR-010 |

### 9.3 仍需处理

| 审查项 | 当前状态 | 下一步 |
| --- | --- | --- |
| ARC-003 | 内部 API 候选已收敛，但字段级 OpenAPI/JSON Schema 未创建 | P2-DB-002 |
| LNG-001 | LangGraph TaskNode/Validation 子图拓扑仍未完全统一 | P2-DB-002 前完成节点表 |
| LNG-002 | DerivationPlan/TruthWitness 尚未定义 | 发布校验 P2-PY-012/013 前完成 |
| LNG-003 | 父任务与阶段超时预算仍缺统一配置 | P2-PY-016 前完成 |
| DATA-003 | 大厅搜索、评分读模型仍缺索引/聚合 | 公开规模前评审 |
| API-002 | 候选 Diff/Artifact 查询与接受/拒绝接口仍缺 | P2-GO-006 / P2-FE-008 前补齐 |
| API-004 | 通知、成就、完整历史和首页高级聚合仍缺 | Open Questions OQ-007 |
| TASK-004 | 多个任务粒度仍偏大 | 执行前按单一聚合/不变量拆分 |
| TASK-006 | PRD `U-10` 与 API `U10` 编号仍可能误读 | 建立全局需求编号规则 |

### 9.4 结论

本次变更形成 Conditional Baseline，没有形成“所有决策已批准”或“所有实现契约
已完成”的结论。Codex 只能开始不依赖 Proposed 决策的 Phase 0 任务；Proposed
决策确认前，不得冻结对应契约或开始联调。

---

## 10. Task 1 复核结论

### 10.1 Task 0 检查

| 项目 | 结论 |
| --- | --- |
| `AGENTS.md` | 完整，已补充 Decision Baseline 和 Proposed 门禁 |
| 重复设计文档 | 未发现重复 PRD；新增架构目录为分层规范，不是重复长文档 |
| `TASKS.md` | 结构可执行；等待房聊天、后台和异步增强依赖倒置已修复 |
| `DESIGN_REVIEW.md` | 历史问题保留；本次追加批准状态，不再把所有候选写成已解决 |
| 明显技术矛盾 | 当前候选文档内部基本一致；7 项 Proposed 尚未获负责人批准 |
| Git | 无提交，所有仓库内容仍未跟踪 |

### 10.2 最终决策状态

| ID | 最终状态 |
| --- | --- |
| ADR-001 Memory 权威写入 | Proposed |
| ADR-002 WebSocket Envelope | Proposed |
| ADR-003 26 字符 ULID | Accepted |
| ADR-004 Event/State Version 分离 | Proposed |
| ADR-005 独立暂停/恢复 | Proposed |
| ADR-006 等待房聊天 | Proposed |
| ADR-007 运营后台公开 MVP 门禁 | Proposed |
| ADR-008 PlayerContext 动态重建 | Proposed |
| ADR-009 P1 异步增强 | Accepted |
| ADR-010 PRD 路径 | Accepted |

### 10.3 开发基线结论

权威状态和门禁以 [DECISION-BASELINE.md](./architecture/DECISION-BASELINE.md) 为准。
当前可开发范围仅包括不依赖 Proposed 决策的 Phase 0 基础设施和工程骨架；
实时通信、事件版本、暂停恢复、等待房聊天、运营后台、PlayerContext 与 Memory
权威写入必须等待负责人批准。

---

## 11. Task 6 游戏引擎实现审查

### 11.1 当前实现边界

Task 6 已实现候选 Go 游戏引擎、原子开局、角色分配、搜证/线索权限、投票、
暂停恢复和结算，并通过真实 MySQL 集成测试。该实现使用现有候选
`state_version/current_event_seq` 字段，不构成 ADR-004 已批准或
`GATE-GAMESTATE` 已通过。

### 11.2 新登记待决项

| 编号 | 问题 | 当前处理 | 影响 |
| --- | --- | --- | --- |
| DQ-08 | `script_phases.entry_condition_json` / `exit_condition_json` 的 DSL 未冻结 | 非空条件对象当前拒绝编译开局 | 复杂阶段条件不能进入运行时 |
| DQ-09 | 剧本杀投票政策没有正式 JSON Schema | 当前要求 `rule_templates.config_json.murder` 显式配置 | 内容生成器/编辑器尚不能稳定产出 |
| DQ-10 | 个人目标、组合胜利和阵营结局规则未结构化 | 当前仅支持“正确指认凶手/无淘汰”候选策略 | 复杂剧本结局不能结算 |
| DQ-11 | 平票后重投与限定候选策略未冻结 | 当前只接受 `tie_policy=no_elimination` | 其他剧本配置会在开局编译时被拒绝 |
| DQ-12 | 剧本角色私密正文的 SecretPayload AAD 约定未正式化 | 当前约定 `script_character:<id>` / `script_clue:<id>` | 内容加密管线需要批准后对齐 |

### 11.3 建议

1. 在 Phase 4 正式发布前冻结阶段条件 DSL 与投票策略 Schema。
2. 将候选 `murder` policy 纳入 `packages/contracts`，但必须等 ADR-004/005
   与内容 Schema 批准后再作为跨语言权威契约。
3. 补齐 `game_viewer_manifests`、Outbox、真实 WebSocket game stream 和
   Redis timer 调度后，再申请 GATE-GAMESTATE/GATE-WS。

---

## 12. Task 7 剧本内容管理实现审查

### 12.1 当前实现边界

Task 7 已实现 `scripts` Draft 创建、章节/阶段/角色/线索/结局/资产/搜证点对象 CRUD、
版本克隆、结构化校验、私有或邀请渠道发布、下架，以及运行时发布版本 Loader。
真实 MySQL 集成测试验证了发布冻结、新版本不影响旧局、下架后新开局被拒绝、历史局仍可
按绑定版本加载。

发布校验会同步执行 Go 引擎可加载性预检。以下未批准协议一旦以非空内容出现，会返回
`state=unsupported` 并阻断发布：

- `script_phases.entry_condition_json` / `exit_condition_json`。
- `script_endings.conditions_json` 的组合结局条件。
- 当前引擎不支持的投票策略、平局重投或组合胜负规则。

### 12.2 新登记待决项

| 编号 | 问题 | 当前处理 | 影响 |
| --- | --- | --- | --- |
| DQ-13 | Task 5 房间创建仍允许 owner 使用自己的 Draft，而 Task 7 开局 Loader 只允许已发布且校验通过的版本 | 房间可创建但开局返回 `CONTENT_UNAVAILABLE` | 作者体验不一致；是否把发布门禁前移到建房需要负责人确认 |
| DQ-14 | 公开渠道审核后台尚未批准/实现 | S18 `channel=public` 返回 `MODERATION_REQUIRED` | 公开 MVP 仍不能通过本任务发布 |
| DQ-15 | 角色私密本、私密线索的 SecretPayload AAD 未批准 | 内容 API 拒绝写 `private_profile_ref` / `private_text_ref`，Loader 只返回引用 ID | 可发布纯公开内容，不能完整承载私密角色内容 |
| DQ-16 | 章节、结局、资产表是 Task 7 为产品对象补齐的最小承载结构 | 已迁移 `script_chapters`、`script_endings`、`script_assets` 与阶段 `chapter_id` | 需负责人确认这些表作为后续正式基线；不是对条件 DSL 的批准 |
| DQ-17 | Reveal gate、场景连接、人物关系等既有设计对象尚未纳入本轮对象 API 和版本深拷贝 | Task 7 只复制章节/阶段/角色/线索/结局/资产/搜证点；旧 `reveal_gate_id` 不跨版本复制 | 含复杂揭示门控或关系图的剧本在补齐前不能视为完整可编辑/可克隆 |

### 12.3 建议

1. 负责人确认是否把“只能选择已发布且校验通过的版本建房”合并到房间创建校验；
   Task 7 未擅自修改 Task 5 已完成业务规则。
2. 公开发布前完成运营审核 ADR、审核 API、审计和 UI 门禁。
3. 在冻结阶段/结局 DSL、投票策略和 SecretPayload AAD 后，再扩展内容编辑器和
   Python 剧本生成输出；当前 Contracts 中新增内容管理类型仍保持 `proposed`。

---

## 13. Task 8 Python Agent Runtime 实现审查

### 13.1 当前实现边界

Task 8 已实现通用 LangGraph Agent Loop、Provider 抽象、Mock 与
OpenAI-compatible Provider、显式 Tool Registry、Context Builder、短期 Memory
端口、进程内 Trace、内部 Agent API 和最小 Go InternalAIClient。

Demo 流程：

```text
AgentRequest
→ 服务认证与幂等检查
→ 授权 Context 裁剪
→ LangGraph model
→ 注册工具调用
→ 工具结果回填
→ model 生成 final output
→ Pydantic 结构化校验
→ AgentResponse
```

Runtime 不直接访问 Go 数据库、不返回权威状态补丁、不决定角色/投票/胜负。当前
只注册 `demo` 任务级输出 Schema；AI DM、NPC 和剧本生成业务节点未实现。

### 13.2 新登记待决项

| 编号 | 问题 | 当前处理 | 影响 |
| --- | --- | --- | --- |
| DQ-18 | AI DM、NPC、规则解释、剧本生成的具体业务 Schema 与字段 ACL 尚未批准 | Registry/兼容性/Active Demo 绑定已批准并验证；业务 Schema 仍逐项待批 | 业务 Agent 不能在没有 Go 校验契约时接入 |
| DQ-19 | 真实 CA、KMS、Redis 多实例和生产 mTLS 部署 | HMAC/Key ID/nonce/ACL/审计/TLS 基础已批准并通过测试级验证；生产基础设施未部署 | 生产门禁仍未通过，测试通过不等于生产验证 |
| DQ-20 | MCP Server 信任模型、工具发现、权限映射和审计尚未批准 | 仅保留 `ToolSpec.mcp_server` 与 `ToolSource` 扩展边界 | 不接真实 MCP Server，不支持任意远程工具发现 |
| DQ-21 | LangGraph Graph Thread 自动恢复与 Production 部署仍未验证 | CI Run `45803587621` 已验证 MySQL/Redis、Multi-Worker、Recovery/Cleanup 和 Backup/Restore | Graph 自动重放、生产 KMS、RTO/RPO 和容量仍未验证 |
| DQ-22 | 真实供应商模型与数据使用政策未配置 | D3/D4 默认阻断；Provider 已实现 MockTransport 测试；未声称远程调用通过 | 上线前需要供应商合同、密钥、配额、数据驻留与内容策略 |

### 13.3 建议

1. 负责人继续确认 AgentRequest/Response 中受 ADR-008 约束的字段、具体业务输出
   Schema 和 DQ-20 工具权限矩阵。
2. 在 AI DM/NPC 接入前，Go 必须提供已裁剪 `AgentRequest.context`，并对结构化输出做第二道权限和规则检查。
3. 生产接入前部署真实 CA/KMS/Redis 多实例并完成 `mv_ai` trace 持久化、
   LangGraph checkpoint、熔断和预算验证。
4. MCP 只在工具来源、权限、结果 Schema 和审计冻结后接入。

### 13.4 Task 9 安全加固进展

Task 9 在不批准 DQ/ADR 的前提下补充了可验证原型：

- 版本化输出 Schema Registry，返回实际 key/version 和无输入回显的校验问题。
- 可选 HMAC-SHA256 请求签名、timestamp、nonce、重放防护和双密钥轮换窗口。
- D3/D4 短期上下文默认不进入模型，需显式配置才允许。
- MCP/外部网络工具默认拒绝，高风险工具需确认，工具结果有大小上限并标记为不可信。
- 进程内 Agent Run 生命周期与 CheckpointReference 元数据原型，不保存完整 Context。

这些原型不能替代 DQ-18 至 DQ-22 的正式批准。详细候选方案和逐项门禁见
[`dq-18-22-ai-runtime-design.md`](./architecture/dq-18-22-ai-runtime-design.md)
与 [`gate-ai-checklist.md`](./architecture/gate-ai-checklist.md)。

---

## 14. Task 11 批准与实施记录

负责人于 2026-11-25 在明确要求 DQ-18/DQ-19 批准后回复“开始执行吧”。本次授权
仅覆盖 DQ-18/DQ-19 候选推荐范围，不包含 DQ-20 至 DQ-22 或 Proposed ADR。

DQ-18 已落地：

- `packages/contracts` 唯一权威源和 142 定义 Registry。
- SemVer、Active Version、追加式版本历史、兼容性基线和漂移检查。
- `DemoAgentOutput@1` 正式 Active 绑定；业务 Agent Schema 仍需逐项批准。
- Replay、Deprecation、Retirement 与保留策略。

DQ-19 已落地：

- mTLS + HMAC `mv-hmac-sha256-v1` 内部安全 profile。
- Key ID、Active/Previous、撤销、Key Provider/KMS 抽象。
- 内存/Redis Nonce Store、原子消费和 Store 故障 fail-closed。
- operation/resource/tenant scope ACL 和安全审计。
- FastAPI TLS 配置和 Go 客户端双向证书握手测试。

生产未验证项：

- 真实 CA 签发/吊销、真实 KMS 和 Redis 多实例部署。
- 生产 mTLS 端到端、证书过期告警和密钥自动轮换。
- 外部 MCP、持久 Checkpoint、真实 Provider 数据政策和正式业务 Schema。

阻塞审计见
[`task-11-approval-blockers.md`](./architecture/task-11-approval-blockers.md)。
