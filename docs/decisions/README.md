# Architecture Decision Records

> 决策基线日期：2026-11-21
>
> 本目录是 MysteryVerse 开发前架构决策的正式索引。ADR 与
> `docs/architecture/system-architecture.md` 共同构成可直接指导实现的技术基线。

| ADR | 决策 | 状态 |
| --- | --- | --- |
| [ADR-001](./ADR-001-memory-authority.md) | Go 是 Memory 元数据唯一权威写入者 | Proposed |
| [ADR-002](./ADR-002-websocket-envelope.md) | 统一 WebSocket Envelope 与消息流语义 | Proposed |
| [ADR-003](./ADR-003-resource-id.md) | 资源 ID 使用纯 26 字符 ULID | Accepted |
| [ADR-004](./ADR-004-event-versioning.md) | Game Event 与 State Version 分离 | Proposed |
| [ADR-005](./ADR-005-game-control.md) | 暂停与恢复使用独立游戏控制命令 | Proposed |
| [ADR-006](./ADR-006-lobby-chat.md) | 等待房聊天进入 MVP | Proposed |
| [ADR-007](./ADR-007-operations-console.md) | 基础运营审核后台是公开 MVP 门禁 | Proposed |
| [ADR-008](./ADR-008-player-context.md) | MVP 每次从可见投影重建 PlayerContext | Proposed |
| [ADR-009](./ADR-009-async-enhancements.md) | 图片生成与 AI 复盘点评为 P1 异步增强 | Accepted |
| [ADR-010](./ADR-010-documentation-structure.md) | 文档结构与权威来源统一 | Accepted |

## 权威顺序

1. 用户在当前任务中的明确指令。
2. 根 `AGENTS.md`。
3. 本目录中状态为 `Accepted` 的 ADR。
4. `docs/architecture/DECISION-BASELINE.md` 的决定状态与门禁。
5. `docs/architecture/`、`docs/api/`、`docs/database/` 下的 Candidate Baseline；
   只有与 Accepted ADR 一致的部分可直接约束实现。
6. 现有长文档：`MysteryVerse-PRD.md`、`TECH_DESIGN.md`、`DATABASE_DESIGN.md`、
   `API_DESIGN.md`、`UI_DESIGN.md`、`TASKS.md`。
7. `DESIGN_REVIEW.md` 中尚未被 ADR 覆盖的审查建议。

`Proposed` ADR 只是待负责人确认的方案，不是已批准规范，不得作为实现依据。
若低优先级文档与 `Accepted` ADR 冲突，以 ADR 为准，并在同一次变更中修正低优先级文档。

