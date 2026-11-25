# ADR-001：Memory 权威写入机制

- 状态：Proposed
- 日期：2026-11-21
- 决策人：项目技术负责人
- 影响范围：Go Runtime、Python AI、`mv_core`、`mv_ai`、NPC Memory、PlayerContext

## 背景

现有文档同时出现 Go/Python 共同写 Memory、Python 直接访问 `npc_memories`
和 Go 唯一权威写入三种表述。这会破坏 schema 最小权限、来源事件校验和
权限审计链。

## 决策

Go 是 Memory 元数据与已接受 MemoryRecord 的唯一权威写入者。

Go 负责：

- 创建、更新、失效和删除已接受的 Memory 元数据。
- 来源事件存在性、可见性和 actor scope 校验。
- 权限、敏感级、摘要泄漏、去重、关系上限和版本控制。
- `supersedes_memory_id`、`dedupe_key`、`record_version` 和审计字段。
- 将摘要正文写入 SecretService 管理的加密载荷。

Python 负责：

- AI 推理、摘要和 `MemoryCandidate` 生成。
- 通过 Go 内部 API 提交候选。
- 读取 Go 授权裁剪后的 Memory View。

Python 不得直接写 `mv_core.npc_memories`、`npc_relationships`、
`secret_payloads` 或任何 Go 权威表。非权威候选记录可以保存在 `mv_ai`，
但不得成为游戏事实来源。

## MemoryCandidate 最小契约

```json
{
  "candidate_id": "01J9Z3K4M7Q2R8T5V6W1X30000",
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X40000",
  "game_npc_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "owner_actor_scope": "npc:01J9Z3K4M7Q2R8T5V6W1X50000",
  "memory_type": "promise",
  "authority_class": "fact",
  "summary": "玩家承诺帮助寻找失踪的弟弟",
  "source_event_ids": ["01J9Z3K4M7Q2R8T5V6W1X60000"],
  "confidence": 0.93,
  "importance": 80,
  "proposed_expiry": "game_end",
  "dedupe_key": "promise:missing-brother",
  "run_id": "01J9Z3K4M7Q2R8T5V6W1X70000",
  "schema_version": 1
}
```

字段约束：

- `authority_class` 只能是 `fact|player_claim|inference|rumor`。
- `summary` 由服务端执行长度、权限和泄漏检查。
- `source_event_ids` 必须属于当前 game，且在 NPC 可见范围。
- `owner_actor_scope` 必须与目标 NPC 的可读范围一致。
- `schema_version` 是整数契约版本，由 `packages/contracts` 冻结。

## 提交流程

1. Python 生成候选并在本地执行 Pydantic/Schema 校验。
2. Python 调用 Go `POST /internal/v1/ai/memory-candidates`。
3. 请求携带服务凭证、`run_id`、`trace_id` 和 `Idempotency-Key`。
4. Go 校验 actor scope、来源事件、秘密泄漏、重复、关系上限和版本。
5. Go 在同一事务中写候选决策、Memory 元数据、加密摘要引用、审计和 Outbox。
6. Go 返回 `accepted|duplicate|rejected|conflict` 的稳定结果。
7. Python 只更新非权威 run 状态，不重试语义拒绝。

## 失败处理

- `VALIDATION_ERROR`：候选不合法，不写权威数据。
- `FORBIDDEN_VIEW`：来源或 actor scope 越权，不重试。
- `IDEMPOTENCY_CONFLICT`：同键不同 payload，不重试。
- `DUPLICATE_CANDIDATE`：返回现有接受结果。
- `VERSION_CONFLICT`：重新读取授权上下文后生成新候选。
- `DEPENDENCY_ERROR`：可使用同一幂等键有限重试。

## 影响

- Memory 元数据只有一条写入和审计链。
- Python 不再需要 `mv_core` 写权限。
- NPC/RPG 的 Memory 任务必须等待 Go 权威接口与 schema 冻结。
- MVP 不持久化 PlayerContext；短期工作记忆仍是非权威缓存。

## 关联文档

- [Memory 架构](../architecture/memory-architecture.md)
- [内部 AI API](../api/ai-internal-api.md)
- [Schema 设计](../database/schema-design.md)
- `TECH_DESIGN.md` §10
- `DATABASE_DESIGN.md` §6.4

