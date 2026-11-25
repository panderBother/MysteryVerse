# 数据库 Schema 设计基线

> 状态：Candidate Baseline，依赖 Proposed ADR
> 数据库：MySQL 8.0.30+
> 依据：ADR-001、ADR-003、ADR-004、ADR-006、ADR-007、ADR-008

完整字段以 `docs/DATABASE_DESIGN.md` 为准。本文件冻结命名、权威 schema、
新增 MVP 表和数据不变量。

## 1. Schema 所有权

| Schema | Owner | 内容 |
| --- | --- | --- |
| `mv_core` | Go | 用户、内容、房间、游戏、事件、Memory 元数据、社区、运营 |
| `mv_ai` | Python | Prompt、Agent Run、Step、Usage、Checkpoint、非权威候选 |
| `mv_audit` | 安全/Worker | 不可变审计归档和事故快照引用 |

账号必须最小权限：

- Python 账号无 `mv_core` 写权限。
- SecretService 是唯一可解密 `secret_payloads` 的 core 路径。
- Admin 使用独立受限查询，不继承 Runtime 权限。

## 2. ID 与标准列

- 业务主键：`CHAR(26) CHARACTER SET ascii COLLATE ascii_bin`。
- 值：纯 ULID，不带类型前缀。
- 时间：UTC `DATETIME(3)`。
- 哈希：`BINARY(32)` SHA-256。
- 金额：`DECIMAL(18,8)`。
- 状态：短 `VARCHAR` + `CHECK`。
- JSON：必须带 `schema_version`，写入前 Schema 校验。
- 默认外键 `ON DELETE RESTRICT`；不使用大范围生产 `CASCADE`。

## 3. 领域表

| 领域 | 表 |
| --- | --- |
| 用户 | `users`、`auth_identities`、`user_sessions`、`user_profiles` |
| 请求控制 | `idempotency_records` |
| 社交 | `friendships`、`user_blocks`、`user_statistics` |
| 内容 | `scripts`、`script_versions`、`story_bible_versions`、`story_facts` |
| 结构化剧本 | `script_characters`、`script_clues`、`script_phases`、`script_tasks` |
| 工坊 | `source_documents`、`generation_jobs`、`generation_job_steps`、`generation_artifacts` |
| 房间 | `game_rooms`、`room_members`、`room_events`、`room_messages`、`invitation_tokens` |
| 游戏 | `game_sessions`、`game_players`、`game_roles`、`game_states` |
| 事件 | `game_events`、`game_intents`、`game_messages`、`outbox_events` |
| 模式 | `game_votes`、`game_search_claims`、`game_night_actions`、`game_world_variables` |
| 快照 | `game_snapshots`、`game_viewer_manifests` |
| 结果 | `game_results`、`replay_reports`、`replay_annotations` |
| Memory | `game_npcs`、`npc_relationships`、`npc_memories`、`npc_memory_snapshots` |
| AI | `prompt_versions`、`agent_runs`、`agent_steps`、`agent_messages`、`llm_usage_records` |
| 运营 | `reports`、`moderation_cases`、`user_sanctions`、`audit_logs`、`incidents` |

不建立：

- 独立 `projects` 与 `scripts` 并存。
- `game_logs` 复制 `game_events`。
- 按 viewer 全量展开的事件副本。
- 只有 `id + JSON` 的通用业务表。

## 4. 新增 MVP 表：`invitation_tokens`

位于 `mv_core`。

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `room_id` | `CHAR(26)` | FK，IX | 目标房间 |
| `created_by` | `CHAR(26)` | FK users | 创建者 |
| `token_hash` | `BINARY(32)` | UQ | 仅保存 token hash |
| `purpose` | `VARCHAR(24)` | CK | `room_invite` |
| `max_uses` | `INT UNSIGNED` | 默认 1 | 最大兑换次数 |
| `use_count` | `INT UNSIGNED` | 默认 0 | 已兑换次数 |
| `expires_at` | `DATETIME(3)` | IX | 过期时间 |
| `revoked_at` | `DATETIME(3)` | NULL | 撤销时间 |
| `invitee_rule_json` | `JSON` | NULL | 账号范围等受限规则 |

索引：

- `UNIQUE(token_hash)`
- `INDEX(room_id, expires_at)`
- `INDEX(created_by, created_at)`

兑换必须使用条件更新或行锁保证 `use_count < max_uses`。token 明文不落库、
不写普通日志。

## 5. 新增 MVP 表：`room_messages`

位于 `mv_core`，独立于 `room_events` 和 `game_events`。

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `room_id` | `CHAR(26)` | FK，UQ(room_id,chat_seq) | 房间 |
| `chat_seq` | `BIGINT UNSIGNED` | UQ(room_id,chat_seq) | 单房间聊天序号 |
| `sender_user_id` | `CHAR(26)` | FK，IX | 发送者 |
| `client_message_id` | `CHAR(26)` | UQ(room_id,sender_user_id,client_message_id) | 幂等 |
| `message_type` | `VARCHAR(16)` | CK | MVP 为 `text` |
| `content_text` | `TEXT` | 长度约束 | D1 正文 |
| `content_hash` | `BINARY(32)` |  | 完整性 |
| `moderation_status` | `VARCHAR(20)` | CK，IX | `pending|allowed|rejected|redacted` |
| `deleted_at` | `DATETIME(3)` | NULL | 软删除 |
| `sent_at` | `DATETIME(3)` | IX | 权威发送时间 |

`chat_seq` 与 `room_events.seq`、`game_events.seq` 完全独立。

## 6. 新增运营表：`user_sanctions`

位于 `mv_core`。

| 字段 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| `user_id` | `CHAR(26)` | FK，IX | 被处置用户 |
| `sanction_type` | `VARCHAR(20)` | CK | `mute|suspend` |
| `scope` | `VARCHAR(32)` | IX | `platform|room` 等 |
| `scope_ref_id` | `CHAR(26)` | NULL | 房间等范围 |
| `reason_code` | `VARCHAR(64)` | IX | 稳定原因 |
| `reason_text` | `VARCHAR(500)` |  | 脱敏说明 |
| `created_by` | `CHAR(26)` | FK | 管理员 |
| `starts_at` | `DATETIME(3)` |  | 生效 |
| `expires_at` | `DATETIME(3)` | NULL，IX | 失效 |
| `revoked_at` | `DATETIME(3)` | NULL | 撤销 |
| `revoked_by` | `CHAR(26)` | NULL | 撤销者 |

所有创建/撤销写 `audit_logs` 和 Outbox。

## 7. AI 候选表：`mv_ai.memory_candidates`

非权威候选记录：

| 字段 | 说明 |
| --- | --- |
| `id` | Python 生成的 ULID |
| `run_id` | AI Run |
| `game_id`、`game_npc_id` | 逻辑作用域，无 core 物理 FK |
| `candidate_hash` | 规范化请求 hash |
| `payload_reference` | 加密候选引用 |
| `schema_version` | 整数 |
| `decision` | `pending|accepted|duplicate|rejected|conflict` |
| `core_memory_id` | Go 接受后的逻辑引用 |
| `decision_reason` | 稳定码 |
| `created_at`、`decided_at` | UTC |

Go 接受后的 `npc_memories` 才是权威 MemoryRecord。

## 8. 进度序号

`generation_jobs` 需要一个持久 `progress_seq BIGINT UNSIGNED DEFAULT 0`，
用于 `ai_progress` 的流序号和重连快照。最终保留时长由 Open Question OQ-005
决定。

## 9. 核心不变量

1. 一房最多一局。
2. `game_events(game_id,seq)` 连续且唯一。
3. `room_events(room_id,seq)` 连续且唯一。
4. `room_messages(room_id,chat_seq)` 连续且唯一。
5. `game_intents(game_id,idempotency_key)` 唯一。
6. 相同幂等键不同 payload 不产生新事件。
7. 一局最多一条 `game_results`。
8. 状态事件后 `game_states.last_event_seq = current_event_seq`。
9. 纯消息事件后 `game_states.last_event_seq <= current_event_seq`。
10. 独占线索最多一个有效 claim。
11. 每局每人每 ballot type 最多一个当前有效投票。
12. Memory 查询必须带 game、NPC 和 actor scope。
13. Python 不能读取 `mv_core.secret_payloads`。
14. 公开版本发布后不可原地修改。

## 10. 迁移

- `goose` 版本化迁移。
- up/down、预检查、锁时长和回滚说明。
- 关键非空列使用 expand-contract。
- 迁移在影子库和 MySQL 8.0.30+ 验证。
- 大表变化不在应用启动时自动执行。

