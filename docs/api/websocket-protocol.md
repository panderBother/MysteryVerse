# WebSocket 协议

> 状态：Candidate Baseline
> 协议：`mvws.v1`
> 依据：ADR-002、ADR-003、ADR-004、ADR-006

## 1. 连接

```text
POST /api/v1/realtime/tickets
wss://api.mysteryverse.example/ws/v1
```

一次性 ticket 绑定 `user_id + room_id + purpose + nonce`，有效期建议 30 秒，
只能成功握手一次。浏览器不得把 ticket 放入 URL 或 LocalStorage。

握手必须校验：

- Session 与账号状态。
- Origin。
- `mvws.v1` 子协议。
- ticket 有效期和 nonce。
- 房间成员和可选游戏席位。

## 2. 服务端 Envelope

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X70000",
  "event_type": "game.clue_unlocked",
  "stream": "game",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "seq": 185,
  "schema_version": 1,
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "timestamp": "2026-11-21T10:05:00Z",
  "payload": {
    "clue_id": "01J9Z3K4M7Q2R8T5V6W1X90000",
    "title": "船票残片",
    "scope": "player"
  }
}
```

| 字段 | 规则 |
| --- | --- |
| `event_id` | 全局唯一纯 ULID，客户端首选去重键 |
| `event_type` | 稳定事件名 |
| `stream` | `system`、`room`、`room_chat`、`game`、`ai_progress` 或 `ai_stream` |
| `stream_id` | 对应流实例纯 ULID |
| `seq` | 同 `stream + stream_id` 内严格递增 |
| `schema_version` | 正整数载荷契约版本 |
| `game_id` | 游戏级事件必填，其他事件不得伪造 |
| `timestamp` | UTC RFC 3339 |
| `payload` | 已授权 viewer 投影 |

## 3. Stream 定义

| stream | stream_id | seq | game_id |
| --- | --- | --- | --- |
| `system` | `connection_id` | 连接控制序号 | 无 |
| `room` | `room_id` | `room_events.seq` | 无 |
| `room_chat` | `room_id` | `room_messages.chat_seq` | 无 |
| `game` | `game_id` | `game_events.seq` | 必填 |
| `ai_progress` | `generation_job_id` | 任务进度序号 | 生成任务无 |
| `ai_stream` | `ai_run_id` | Run 临时流序号 | 游戏内必填 |

不同 stream 的 `seq` 不可比较。等待房消息不得伪造 `game_id`。

## 4. 客户端消息

客户端消息使用请求信封，不伪造服务端权威字段：

```json
{
  "request_id": "01J9Z3K4M7Q2R8T5V6W1X20000",
  "event_type": "game.subscribe",
  "schema_version": 1,
  "payload": {
    "game_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
    "last_game_seq": 184,
    "last_state_version": 17
  }
}
```

允许：

- `room.subscribe`
- `room_chat.subscribe`
- `game.subscribe`
- `resync.request`
- `heartbeat.ping`
- `message.ack`

禁止：

- 客户端提供 `event_id`、`seq`、`visibility` 或 `state_version` 作为权威值。
- 订阅未加入房间或未绑定席位的游戏。
- 自行生成系统消息类型。

## 5. 生命周期消息

`server.ready`：

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X30000",
  "event_type": "server.ready",
  "stream": "system",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X40000",
  "seq": 1,
  "schema_version": 1,
  "timestamp": "2026-11-21T10:00:00Z",
  "payload": {
    "user_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
    "room_seq": 12,
    "room_chat_seq": 18,
    "game_seq": 184,
    "state_version": 17,
    "heartbeat_interval_ms": 25000
  }
}
```

认证成功前不得发送任何房间或游戏数据。

## 6. 快照与增量

- `room.snapshot`
- `room_chat.snapshot`
- `game.snapshot`
- 对应增量事件

快照必须包含：

- 当前 stream ID。
- 覆盖的 seq。
- 游戏快照的 `state_version`。
- viewer capabilities 和 available actions。
- 服务端时间。

大快照优先通过 HTTP 获取。单条 WS 消息不得携带无界历史。

## 7. 去重、乱序和缺口

1. 同一 `event_id` 只应用一次。
2. 同一 `stream + stream_id` 中 `seq <= lastSeq` 的消息忽略。
3. 收到大于 `lastSeq + 1` 的序号时停止应用，发送 `resync.request`。
4. `room`、`room_chat`、`game` 分别保存 cursor。
5. 快照整体替换对应投影，不做双向合并。
6. AI 流片段只展示，不作为权威事件；正式消息到达后替换临时文本。

## 8. 可见性

- 服务端在发送前执行对象 ACL、字段 ACL 和 reveal gate。
- 私聊通过用户定向频道。
- 阵营消息通过角色组定向频道。
- `room_chat` 只发送给有效房间成员。
- 公共事件不得携带未授权角色本、密文、Prompt 或完整身份表。

## 9. AI 流

- `ai_progress` 用于生成任务节点进度或 SSE 映射。
- `ai_stream` 用于游戏内 DM/NPC 展示片段。
- 两者都不递增 `state_version` 或 `game_events.seq`。
- 断线后不保证重放 AI chunk；客户端等待正式持久消息。
- Go 必须把内部 `ai.node.*` 映射为公共契约。

## 10. 错误

服务端错误使用统一错误对象：

- `PROTOCOL_UNSUPPORTED`
- `AUTH_REQUIRED`
- `NOT_ROOM_MEMBER`
- `NOT_GAME_PLAYER`
- `FORBIDDEN_VIEW`
- `EVENT_GAP_TOO_LARGE`
- `RATE_LIMITED`
- `DEPENDENCY_ERROR`

错误不得包含堆栈、SQL、Prompt 或供应商原文。

## 11. 测试

- ticket 单次使用和过期。
- Origin、子协议和 Session 撤销。
- 各 stream 独立 seq。
- 重复、乱序和缺口恢复。
- 快照替换与权限变化。
- 私密事件定向投递。
- 慢客户端背压和限流。
- AI 流不进入权威事件。

