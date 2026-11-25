# ADR-003：资源 ID 规范

- 状态：Accepted
- 日期：2026-11-21
- 影响范围：MySQL、API、WebSocket、对象存储、日志、代码生成

## 决策

所有持久资源主键和跨服务资源 ID 使用纯 26 字符 ULID：

```text
[0-9A-HJKMNP-TV-Z]{26}
```

规则：

- 数据库主键使用 `CHAR(26) CHARACTER SET ascii COLLATE ascii_bin`。
- 网络契约不接收或返回 `game_`、`room_`、`script_`、`evt_` 等类型前缀。
- 资源类型由字段名、Schema、API 路径和业务上下文表达。
- 前端可显示 `房间 01J...`、`事件 01J...` 等类型标签，但标签不属于 ID。
- 客户端不得生成权威资源 ID。
- 每个 schema owner 使用统一 ULID 库生成自己拥有的持久记录。
  Go 权威资源由 Go `platform/id` 生成；`mv_ai` 资源由 Python 使用同一规范生成。
- 外部供应商 ID、request ID 和 trace ID 如果不是业务资源主键，可以保持供应商格式，
  但不得写入 `CHAR(26)` 业务主键列。

## 唯一性与排序

- ULID 由 48 位毫秒时间戳和 80 位加密随机部分组成。
- 同一进程在相同毫秒内必须使用单调 ULID 生成策略，避免顺序倒挂。
- 字典序近似时间序，但业务排序必须使用明确时间列或 `seq`，不得把 ID 排序当权威事件顺序。
- 唯一性由生成器质量、数据库主键、唯一约束和冲突重试共同保证。

## 示例

正确：

```json
{
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "room_id": "01J9Z3K4M7Q2R8T5V6W1X20000",
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X30000"
}
```

禁止：

```json
{
  "game_id": "game_01J9Z3K4M7Q2R8T5V6W1X1",
  "event_id": "evt_01J9Z3K4M7Q2R8T5V6W1X3"
}
```

## 影响

- 现有带前缀示例必须标记为历史非规范示例，并逐步替换。
- OpenAPI/JSON Schema 统一使用长度 26 和 ULID pattern。
- 日志字段名保留资源语义，值统一为纯 ULID。

## 关联文档

- [Schema 设计](../database/schema-design.md)
- `API_DESIGN.md` §1
- `DATABASE_DESIGN.md` §1.2、§5.1

