# 事件架构

> 状态：Candidate Baseline，依赖 Proposed ADR
> 依据：ADR-002、ADR-004、ADR-006

## 1. 事件类别

| 类别 | 持久化 | Seq 空间 | 改变 state_version |
| --- | --- | --- | --- |
| 房间状态事件 | `room_events` | `room_events.seq` | 否，独立房间状态 |
| 等待房聊天 | `room_messages` | `room_messages.chat_seq` | 否 |
| 游戏权威事件 | `game_events` | `game_events.seq` | 视事件类型 |
| AI 任务进度 | task projection / job sequence | `ai_progress` | 否 |
| AI 流片段 | 非权威临时流 | `ai_stream` run seq | 否 |

## 2. Game Event 与 State Version

- `event_seq` 覆盖所有权威游戏事件。
- `state_version` 只覆盖规则状态变更。
- 聊天、私聊、DM 正式消息、NPC 正式叙事可进入 `game_events`，
  但只递增 `event_seq`。
- 投票、阶段、暂停、恢复、线索、技能、角色状态和结算同时递增两者。
- `game_sessions.current_event_seq` 始终表示最后一条 game event。
- `game_states.last_event_seq` 表示最后一条实际改变当前状态的事件。
- 纯消息事务后允许 `last_event_seq < current_event_seq`。

## 3. 事务

状态事件事务：

```text
game_sessions lock
→ idempotency
→ expected_version
→ engine validate/apply
→ events
→ current state
→ state_version + event_seq + state_hash
→ intent result
→ outbox
→ commit
```

消息事件事务：

```text
permission/stage/mute
→ idempotency
→ allocate event_seq
→ game_event + game_message
→ current_event_seq
→ outbox
→ commit
```

禁止：

- 事务内调用模型、对象存储或外部 HTTP。
- 先广播后提交。
- 消息事件修改规则状态。
- 用客户端序号或时间决定权威顺序。

## 4. 可见性

每条事件必须声明：

- `visibility_scope`。
- recipients 或 role group。
- `reveal_gate`。
- `sensitivity`。
- payload 加密引用。

统一事件存储不代表统一公开。投影器必须先 ACL 后构造 DTO。

## 5. 幂等与并发

- 游戏意图：`game_id + idempotency_key`。
- 投票：每轮投票人 ballot type 唯一。
- 聊天：发送者 `client_message_id` 唯一。
- 暂停/恢复：独立 `game_id + idempotency_key`。
- 同一游戏只允许一个事务成功推进同一 `state_version`。

## 6. 重连

客户端保存：

- `last_room_seq`
- `last_room_chat_seq`
- `last_game_seq`
- `last_state_version`

重连时：

1. 重新鉴权。
2. 获取服务端快照。
3. 清空旧投影。
4. 按授权增量补偿。
5. 缺口过大时只使用快照。

AI 流片段不重放；正式 AI 消息从持久事件恢复。

## 7. 回放

- 规则回放使用 package/engine version、snapshot、events 和 result。
- 展示回放只投影已授权事件。
- AI 点评是附加 artifact，不改变回放轨迹。
- 聊天事件是否进入展示回放由 scope 和产品保留策略决定。

## 8. 数据不变量

1. `game_events.game_id + seq` 唯一且连续。
2. `current_event_seq` 与最后事件一致。
3. 状态事件后 `last_event_seq = current_event_seq`。
4. 消息事件不递增 `state_version`。
5. 结算每局最多一条。
6. 事件错误通过补偿事件修正。
7. 私密事件不得出现在公共投影或分享。

## 9. 相关文档

- [事件模型](../database/event-model.md)
- [WebSocket 协议](../api/websocket-protocol.md)
- `TECH_DESIGN.md` §6、§11
- `DATABASE_DESIGN.md` §12

