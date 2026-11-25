# ADR-006：等待房聊天

- 状态：Proposed
- 日期：2026-11-21
- 影响范围：房间 API、房间数据库、WebSocket、UI、内容审核

## 决策

MVP 保留等待房聊天，并使用独立房间消息流：

```text
stream = room_chat
stream_id = room_id
seq = room_messages.chat_seq
```

等待房聊天不得复用正式游戏的 `game_events.seq`，等待房消息不得伪造
`game_id`。

## API

```http
GET  /api/v1/rooms/{room_id}/messages?cursor=<cursor>&limit=30
POST /api/v1/rooms/{room_id}/messages
```

发送请求：

```json
{
  "client_message_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "text": "大家准备好了吗？"
}
```

请求头必须包含 `Idempotency-Key`。响应返回消息、`event_id`、`chat_seq`。

## 数据与权限

- 消息持久化到 `room_messages`。
- `chat_seq` 在 `room_id` 内单流严格递增，与 `room_events.seq` 独立。
- 只有有效房间成员可读取或发送。
- 房主、创作者和运营没有额外读取他人私密内容的默认权限。
- 发送受文字长度、频率、禁言、房间状态和内容审核限制。
- 消息正文属于 D1 房间共享数据；不得进入公开 CDN。

## 生命周期

- 房间关闭后按策略保留或归档，不影响游戏事件、结算和复盘。
- 游戏开始后 `room_chat` 停止；正式游戏聊天使用 game message API。
- 消息删除采用软删除、审核状态或正文清理，不修改事件顺序。

## 非阻塞规则

- 等待房聊天故障不得阻止创建、准备、开始游戏。
- 聊天超时或审核失败不得改变房间状态。
- UI 在聊天不可用时必须保留开始/准备主操作。

## 关联文档

- [WebSocket 协议](../api/websocket-protocol.md)
- [Schema 设计](../database/schema-design.md)
- `API_DESIGN.md` §5
- `UI_DESIGN.md` §5.9

