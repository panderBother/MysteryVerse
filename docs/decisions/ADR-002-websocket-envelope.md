# ADR-002：WebSocket Envelope 统一协议

- 状态：Proposed
- 日期：2026-11-21
- 影响范围：Go Realtime、前端 WS reducer、SSE 映射、内部事件、契约生成

## 决策

所有服务端到客户端实时消息使用统一 Envelope：

```typescript
interface WebSocketEnvelope<T = unknown> {
  event_id: string;
  event_type: string;
  stream: "system" | "room" | "room_chat" | "game" | "ai_progress" | "ai_stream";
  stream_id: string;
  seq: number;
  schema_version: number;
  game_id?: string;
  timestamp: string;
  payload: T;
}
```

字段规则：

- `event_id`：全局唯一纯 26 字符 ULID，是客户端首选去重键。
- `event_type`：稳定事件类型。
- `stream`：消息流类别，不能由客户端任意扩展；新增值必须更新契约。
- `stream_id`：该流实例的 ULID。
- `seq`：在相同 `stream + stream_id` 内严格单调递增。
- `schema_version`：正整数事件载荷契约版本；与 GamePackage 的 SemVer 字符串分离。
- `game_id`：`stream=game|ai_stream` 且事件属于游戏时必填；等待房消息不得伪造。
- `timestamp`：UTC RFC 3339，由服务端写入。
- `payload`：已按 viewer 完成权限投影的 DTO。

## Stream 映射

| stream | stream_id | seq 来源 | game_id |
| --- | --- | --- | --- |
| `system` | `connection_id` | 连接生命周期控制序号 | 无 |
| `room` | `room_id` | `room_events.seq` | 无 |
| `room_chat` | `room_id` | `room_messages.chat_seq` | 无 |
| `game` | `game_id` | `game_events.seq` | 必填 |
| `ai_progress` | `generation_job_id` | 任务进度序号 | 内容生成任务无 |
| `ai_stream` | `ai_run_id` | 单次 AI Run 流片段序号 | 游戏内 AI 必填 |

`room`、`room_chat`、`game` 的序号空间相互独立，不能交叉比较。

## 客户端消息

客户端消息不伪造服务端 Envelope。允许的客户端消息类型由契约定义：

- `room.subscribe`
- `room_chat.subscribe`
- `game.subscribe`
- `resync.request`
- `heartbeat.ping`
- `message.ack`

客户端不得提交 `event_id`、`seq`、`schema_version`、`visibility` 或
`state_version` 来伪造权威事件。

## 重连与补偿

1. 客户端分别保存 `last_room_seq`、`last_room_chat_seq`、`last_game_seq`、
   `last_state_version` 和活动 AI Run 信息。
2. 建立连接后重新校验 Session、ticket、房间成员和游戏席位。
3. 服务端返回快照或授权增量。
4. 某一流的序号缺口停止应用，发送该流的 `resync.request`。
5. 大快照通过 HTTP 投影接口获取，不塞入单条 WebSocket 消息。
6. 客户端清空对应旧投影后应用快照与增量，禁止双向合并。
7. `ai_stream` 不保证断线后重放；最终正式 AI 消息以持久事件为准。

## 可见性与内部事件

- 服务端在发送前执行对象 ACL、字段 ACL、reveal gate 和 viewer 投影。
- 私密事件只进入授权用户或角色组定向频道。
- Python 内部 `ai.node.*` 不得原样进入浏览器；Go 映射为公共 SSE/WS 契约。
- `room_chat` 与 `game` 消息即使承载文本，也不得绕过 scope 过滤。

## 影响

- TECH 的 `messageId/streamId` 与 API 的 `event_id/stream_id` 双命名关闭。
- `session_id` 不再作为 WebSocket 对局标识；统一使用 `game_id`。
- 旧协议示例必须按本 ADR 更新，不得生成双 envelope。

## 关联文档

- [WebSocket 协议](../api/websocket-protocol.md)
- [事件架构](../architecture/event-architecture.md)
- `API_DESIGN.md` §8
- `TECH_DESIGN.md` §11

