# Memory 架构

> 状态：Candidate Baseline，依赖 Proposed ADR
> 依据：ADR-001、ADR-008

## 1. 权威关系

| 数据 | 权威 | AI 用途 |
| --- | --- | --- |
| GameState、关系数值、承诺、任务 | Go | 已授权事实切片 |
| MemoryRecord 元数据 | Go | 授权 Memory View |
| 加密记忆摘要 | Go SecretService | 经授权读取 |
| MemoryCandidate | Python 生成，非权威 | 提交 Go |
| Prompt、Run、Usage、Checkpoint | Python `mv_ai` | 运行时 |
| 短期工作记忆 | Redis/进程内存 | 可重建 |

## 2. 分层

### 2.1 PlayerContext

MVP 每次 Agent 执行从当前可见投影重建，不持久化快照。必须带版本和 actor scope。

### 2.2 短期 Working Memory

- 最近对话回合。
- 当前阶段与可见状态摘要。
- 近期相关事件。
- 当前目标、临时情绪和推理。
- Redis TTL 或进程内存。
- 可随时重建，不是权威事实。

### 2.3 长期 Memory

- NPC 人格背景引用。
- 玩家/NPC 关系历史。
- 重要经历、承诺、交易和传闻。
- 带来源事件、授权类别、置信度和生命周期。
- 进入 Phase 2 后才启用完整 NPC Memory。

### 2.4 Snapshot

长期 Memory Snapshot 为 Phase 2 能力，MVP 不持久化 PlayerContext。
未来快照必须使用独立 ADR，说明作用域、加密、版本和恢复。

## 3. MemoryRecord

权威记录至少包含：

- `memory_id`
- `game_id`、`game_npc_id`
- `owner_actor_scope`
- `authority_class`
- `record_version`
- `summary_secret_payload_id`
- `source_event_ids`
- `dedupe_key`
- `importance`、`confidence`
- `supersedes_memory_id`
- lifecycle 字段
- sensitivity/access scope

查询必须带 `game_id + game_npc_id + actor_scope`。

## 4. 写入流程

1. Python 生成 MemoryCandidate。
2. Python 调用 Go 内部 API。
3. Go 校验来源事件、可见性、泄漏、重复和关系上限。
4. Go 在事务中写加密摘要引用和 Memory 元数据。
5. Go 写审计与 Outbox。
6. Python 读取接受结果，不获得 core 写权限。

## 5. 冲突

- 玩家自述默认为 `player_claim`。
- 模型推断为 `inference`。
- 文档/事件确认的事实为 `fact`。
- 传闻保持来源和低置信度。
- Memory 与 Go 当前状态冲突时以 Go 为准，旧记忆标记过期或待修复。

## 6. 失败

- Candidate Schema 错误：拒绝，不重试。
- 越权/秘密：安全拒绝，不重试。
- 版本冲突：重新读取上下文。
- 依赖故障：同幂等键有限重试。
- 重复：返回现有或合并结果。

## 7. MVP 范围

MVP 需要：

- 每次重建 PlayerContext。
- AI DM 所需短期可见上下文。
- 不允许 NPC 长期记忆进入剧本杀规则。

Phase 2 才实现：

- 长期 NPC Memory。
- 关系演化和承诺追踪。
- Memory Snapshot 与恢复。

## 8. Task 8 当前实现

Python `MemoryStore` 端口当前只有 `InMemoryMemoryStore`：

- 按 actor/game scope 保存有限条短期上下文。
- 进程重启即丢失，可从 Go 授权上下文重建。
- 不保存 MemoryCandidate、MemoryRecord、关系数值或游戏事实。
- 不构成 ADR-001 已批准或 Go Memory 表的旁路。

未来 Redis/MySQL 实现必须遵守 ADR-001、作用域、TTL、加密和清理策略。

