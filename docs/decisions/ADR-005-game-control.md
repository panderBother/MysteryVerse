# ADR-005：游戏暂停与恢复

- 状态：Proposed
- 日期：2026-11-21
- 影响范围：API、Game Runtime、Timer、状态机、WebSocket、审计

## 决策

暂停和恢复是独立 GameSession 控制命令：

```http
POST /api/v1/games/{game_id}/pause
POST /api/v1/games/{game_id}/resume
```

请求：

```json
{
  "expected_version": 12
}
```

请求头：

```http
Idempotency-Key: <unique-key>
```

## 服务端规则

1. 校验 Session、房间成员、房主权限、模式暂停规则和当前 `game_id`。
2. 校验 `expected_version = game_sessions.state_version`。
3. 查询 `game_id + idempotency_key`；同键同 payload 返回原结果。
4. 在同一事务内锁定 `game_sessions`。
5. 检查 `running`/`paused` 状态和 timer 状态。
6. 执行状态机控制转换。
7. 写 `game.paused` 或 `game.resumed` 权威事件。
8. 更新 `state_version`、`current_event_seq`、`state_hash` 和 timer。
9. 写幂等结果、Outbox 和审计。
10. 提交后广播新的投影。

暂停必须保存剩余毫秒；恢复必须重新设置权威 `deadline_at`。客户端计时器
不得决定暂停或恢复。

## 错误行为

- 重复同 payload：返回首次结果，`meta.idempotent_replay=true`。
- 同键不同 payload：`409 IDEMPOTENCY_CONFLICT`。
- 旧版本：`409 STATE_VERSION_STALE`。
- 并发同版本：一个成功，其余 `409 STATE_VERSION_CONFLICT`。
- 当前模式或阶段不可暂停：`409 PAUSE_NOT_ALLOWED`。
- 已在目标状态：合法幂等返回，不产生重复事件。

## 影响

- `rooms/{room_id}/host-actions` 不再承载游戏内 `pause|resume`。
- 房主身份只是授权条件，不得绕过 GameSession 行锁和版本控制。
- 每个暂停/恢复只影响目标游戏，不改变等待房状态。

## 关联文档

- [游戏 API](../api/game-api.md)
- [事件架构](../architecture/event-architecture.md)
- `API_DESIGN.md` §5、R10
- `DATABASE_DESIGN.md` §12.11

