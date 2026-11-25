# ADR-004：Game Event 与 State Version 分离

- 状态：Proposed
- 日期：2026-11-21
- 影响范围：Game Runtime、事件投影、聊天、DM、重连、回放、数据库事务

## 决策

严格区分：

- `event_seq`：单局权威事件序列，所有进入 `game_events` 的事件递增。
- `state_version`：只由规则状态变更递增。

规则：

- 所有权威游戏事件写入 `game_events`。
- 公开聊天、私聊、AI DM 正式消息、NPC 正式叙事和系统公告进入 `game_events`，
  递增 `event_seq`，但不改变规则状态时不得递增 `state_version`。
- 投票、阶段切换、暂停、恢复、角色状态、线索认领、技能和结算等状态变更
  同时递增 `event_seq` 与 `state_version`。
- `game_sessions.current_event_seq` 表示最后一条权威游戏事件。
- `game_states.last_event_seq` 表示最后一条实际应用于当前状态的权威事件；
  纯消息事件不改变该值。
- 纯消息事务之后允许 `game_states.last_event_seq < game_sessions.current_event_seq`。
- 状态变更事务之后两者必须相等。
- `projection_version` 不作为 Envelope 必填字段；客户端以 `state_version`
  和授权快照/事件投影识别状态新鲜度。

## 事务边界

状态变更事务：

1. 锁定 `game_sessions`。
2. 校验幂等和 `expected_version`。
3. 执行 Go 规则引擎。
4. 分配连续事件序号。
5. 追加 `game_events`。
6. 更新当前状态、`state_version`、`current_event_seq`、`last_event_seq` 和 `state_hash`。
7. 写 `game_intents` 与 Outbox。
8. 提交后投影和广播。

纯消息/叙事事务：

1. 锁定必要的 conversation/rate-limit 资源。
2. 校验发送权限、阶段、禁言和可见 scope。
3. 分配连续 `game_events.seq`。
4. 追加消息事件或 `game_messages` 关联。
5. 更新 `current_event_seq`，不递增 `state_version`。
6. 写幂等结果与 Outbox。
7. 提交后只向授权 viewer 投影。

## 可见性

事件存储统一不等于事件内容统一公开。每条事件必须保存：

- `visibility_scope`。
- `visibility_json` 或 recipient scope。
- `reveal_gate`。
- `sensitivity`。
- D2-D4 加密载荷引用。

投影规则：

- 公共聊天只进入 room scope。
- 私聊只进入参与双方或明确授权旁听者。
- AI DM 仅展示已通过安全检查和 Go 确认的正式消息。
- AI 流片段不是权威事件，不进入复盘。
- 内部规则轨迹、Prompt、模型上下文和 trace 不进入玩家事件投影。

## 影响

- “每次成功命令必增 state_version”旧表述废止。
- 聊天和 DM 可以进入权威事件日志，但不得制造状态版本冲突。
- 重连分别使用 `last_game_seq` 与 `last_state_version`。
- Database 不变量改为：状态事件后 `last_event_seq = current_event_seq`；
  消息事件后 `last_event_seq <= current_event_seq`。

## 关联文档

- [事件架构](../architecture/event-architecture.md)
- [事件模型](../database/event-model.md)
- [游戏 API](../api/game-api.md)
- `TECH_DESIGN.md` §6
- `DATABASE_DESIGN.md` §12

