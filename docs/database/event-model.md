# 事件模型

> 状态：Candidate Baseline，依赖 Proposed ADR
> 依据：ADR-002、ADR-004、ADR-006

## 1. 表职责

| 表 | 作用 | Seq |
| --- | --- | --- |
| `room_events` | 房间状态、准备、房主变化 | `room_id + seq` |
| `room_messages` | 等待房聊天 | `room_id + chat_seq` |
| `game_events` | 全部权威游戏事件 | `game_id + seq` |
| `game_messages` | 游戏聊天与正式 AI 消息正文/索引 | 关联 `game_event_id` |
| `outbox_events` | 提交后可靠发布 | 消息 ID 幂等 |
| `audit_logs` | 后台和敏感操作 | 追加写 |

不得建立 `game_logs` 复制 `game_events`。

## 2. `game_events`

最小字段：

| 字段 | 说明 |
| --- | --- |
| `id` | 纯 ULID |
| `game_id` | 对局 |
| `seq` | 单局连续递增 |
| `event_type` | 稳定名称 |
| `event_class` | `state|message|narrative|system|incident` |
| `actor_type`、`actor_id` | 操作者 |
| `causation_id` | 起因 ID |
| `visibility_scope` | public/room/role/role_group/player/runtime |
| `visibility_json` | 定向接收者 |
| `reveal_gate_id` | 揭示门控 |
| `sensitivity` | D0-D4 |
| `payload_json` | 非秘密载荷 |
| `secret_payload_id` | 私密载荷引用 |
| `schema_version` | 整数 |
| `state_version_before` | 仅状态事件 |
| `state_version_after` | 仅状态事件 |
| `payload_hash` | 完整性 |
| `occurred_at` | 服务端时间 |

约束：

- `UNIQUE(game_id,seq)`
- 状态事件的 `after = before + 1`
- 消息事件不填写增一的状态版本，或两者相同
- 事件不可原地修改

## 3. Event Class

| class | 示例 | state_version |
| --- | --- | --- |
| `state` | phase_changed、vote_resolved、paused、resumed | 递增 |
| `message` | public_chat、private_chat | 不递增 |
| `narrative` | AI DM 正式发言、NPC 正式台词 | 不递增 |
| `system` | 公开系统公告、阶段模板播报 | 不递增 |
| `incident` | 冻结、恢复、终止审计 | 视控制动作 |

## 4. 消息事件

聊天/叙事统一进入 `game_events` 时：

- 必须递增 `event_seq`。
- 不得因文本内容改变规则状态。
- 必须通过 scope 投影。
- 私聊正文进入 `game_messages.secret_payload_id` 或加密载荷。
- 客户端系统消息类型被禁止。

`game_messages` 保存业务查询所需正文/索引；`game_events` 保存权威顺序和可见性。
两者在同一事务中写入。

## 5. 序号分配

1. 锁定 `game_sessions`。
2. 读取 `current_event_seq`。
3. 本事务事件从 `current_event_seq + 1` 连续分配。
4. 更新 `current_event_seq`。
5. 状态变更时更新 `state_version`、`state_hash` 和 `last_event_seq`。
6. 纯消息只更新 `current_event_seq`。

多事件事务必须使用同一锁和连续范围，不留空洞。

## 6. 可见性

- `public`：所有有权查看该局公开状态的主体。
- `room`：有效房间成员。
- `role`：指定角色。
- `role_group`：阵营或授权组。
- `player`：指定玩家。
- `runtime`：Go Runtime/极小权限 Worker，不发给浏览器。

服务端先权限过滤，再投影 payload。事件存储中的 recipient JSON 不得直接返回客户端。

## 7. 幂等

- `game_intents` 保存最终结果和事件范围。
- 同键同 hash 返回原事件范围。
- 同键不同 hash 返回冲突。
- 失败校验不占用成功事件序号。
- Outbox 允许重复投递，消费者幂等。

## 8. 恢复

客户端分别恢复 room、room chat、game 和状态版本：

```text
room_seq
room_chat_seq
game_seq
state_version
```

差值过大时只返回快照。事件补偿必须重新投影，不能直接发送历史原始 payload。

## 9. 回放

- 规则回放使用 snapshots + state events。
- 展示回放使用授权事件。
- 纯消息事件是否出现在展示回放取决于产品保留策略。
- AI commentary 是 independent artifact，不能改变状态或结果。

## 10. 测试

- 连续 seq、唯一性和并发。
- 状态/消息版本分离。
- 私聊不会出现在公共投影。
- 事件缺口、重复和乱序。
- snapshot + delta 重建。
- result 唯一与 replay 一致性。

