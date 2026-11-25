# 游戏 API 基线

> 状态：Candidate Baseline
> 根路径：`/api/v1`
> 依据：ADR-003、ADR-004、ADR-005、ADR-009

完整字段展开以 `docs/API_DESIGN.md` 为准。本文件冻结游戏命令的共同语义。

Task 6 已实现候选范围：R09、G01/G02/G06、G09/G10、M01-M05、M07/M08。
ADR-004 与 ADR-005 仍为 Proposed，因此本轮实现不构成正式 `GATE-GAMESTATE`
批准。G03-G05/G07/G08、真实 WebSocket 与 AI DM 不在本轮实现范围。

## 1. 通用请求

游戏状态写操作必须带：

```http
Idempotency-Key: <unique-key>
Content-Type: application/json; charset=utf-8
```

```json
{
  "expected_version": 12
}
```

规则：

- `expected_version` 与服务端 `state_version` 比较。
- 同幂等键同 payload 返回原结果。
- 同幂等键不同 payload 返回 `IDEMPOTENCY_CONFLICT`。
- 版本落后返回 `STATE_VERSION_STALE`。
- 并发推进返回 `STATE_VERSION_CONFLICT`。
- 客户端不提交状态、事件、身份表或结果。

## 2. 查询与重连

```text
GET  /api/v1/games/{game_id}/projection
GET  /api/v1/games/{game_id}/events?after_seq=<seq>
POST /api/v1/games/{game_id}/reconnect
```

投影响应必须包含：

- `game_id`
- `game_type`
- `state_version`
- `event_seq`
- `phase`
- `public_state`
- `private_state`
- `available_actions`
- `viewer_capabilities`

权限根据当前 viewer 实时计算，不能持久化跨用户投影副本。

## 3. 暂停与恢复

```text
POST /api/v1/games/{game_id}/pause
POST /api/v1/games/{game_id}/resume
```

行为：

- Host + 模式规则授权。
- 独立 GameSession 行锁。
- 写权威 `game.paused` / `game.resumed` 事件。
- 递增 `event_seq` 和 `state_version`。
- 保存/恢复权威 timer。
- 幂等与版本冲突遵循第 1 节。

`rooms/{room_id}/host-actions` 不再在游戏内执行 `pause|resume`。

## 4. 游戏消息

```text
GET  /api/v1/games/{game_id}/messages
POST /api/v1/games/{game_id}/messages
```

发送示例：

```json
{
  "client_message_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "message_type": "private",
  "recipient_player_ids": ["01J9Z3K4M7Q2R8T5V6W1X20000"],
  "text": "你昨晚在哪里？",
  "expected_version": 18
}
```

规则：

- `message_type` 只允许 `public|private`。
- 客户端不得提交 `system` 或 `system_note`。
- 正式系统消息只由 Go 或授权运营动作生成。
- 消息进入 `game_events` 时递增 `event_seq`，不改变规则状态时
  不递增 `state_version`。
- 私聊只投影给参与双方或明确授权旁听者。

## 5. 模式接口

保留模式专用路径，不使用万能游戏动作：

- 剧本杀：角色、阅读、搜证、线索、质询、投票。
- 狼人杀：身份、夜间行动、发言、投票、技能、遗言。
- RPG：场景、移动、探索、自由行动、NPC、任务、物品、选择。

所有模式写操作必须经过同一 Go Intent 链：

```text
auth
→ room/seat/role
→ idempotency
→ expected_version
→ engine validate
→ transaction
→ projection
```

## 6. AI DM

```text
POST /api/v1/games/{game_id}/ai-dm/turns
```

规则：

- Go 构建授权上下文。
- Python 返回结构化候选。
- Go 再执行 Schema、规则和安全检查。
- 只有 Go 确认后的正式消息进入 `game_events`。
- AI 失败可返回模板降级；不影响规则动作。

## 7. 结果与复盘

```text
GET /api/v1/games/{game_id}/result
GET /api/v1/games/{game_id}/replay
```

- 规则结果由 Go 生成且不可变。
- 结构化时间线来自权威事件。
- AI 点评是 P1 异步增强，失败不阻塞结果。
- 分享和复盘必须重新执行视角权限。

## 8. 错误码基线

- `GAME_NOT_FOUND`
- `GAME_NOT_STARTED`
- `GAME_ALREADY_ENDED`
- `STATE_VERSION_STALE`
- `STATE_VERSION_CONFLICT`
- `IDEMPOTENCY_CONFLICT`
- `PAUSE_NOT_ALLOWED`
- `PHASE_NOT_ALLOWED`
- `ACTION_NOT_ALLOWED`
- `FORBIDDEN_VIEW`
- `AI_TIMEOUT`
- `PROVIDER_UNAVAILABLE`

