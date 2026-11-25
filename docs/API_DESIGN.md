# MysteryVerse API 设计规范

> 文档版本：v1.0
> 状态：前后端通信契约评审基线
> 上游依据：`docs/MysteryVerse-PRD.md`、`docs/TECH_DESIGN.md`、`docs/DATABASE_DESIGN.md`
> API 根路径：`/api/v1`
> 外部 JSON：`snake_case`
> 时间格式：UTC RFC 3339，如 `2026-11-20T20:00:00Z`
> 实时协议：WebSocket `wss://.../ws/v1`

> **2026-11-21 架构基线补充**：ADR-003、ADR-009、ADR-010 为 Accepted；
> ADR-002、ADR-004、ADR-005、ADR-006、ADR-007 为 Proposed。
> Proposed 内容在负责人确认前不得冻结为 API 契约。

---

## 0. 设计结论

### 0.1 架构边界

1. 浏览器只调用 Go 公共 API 和 WebSocket；Python AI 服务不暴露公网。
2. Go Runtime 是房间、游戏状态、身份、投票、线索和结算的唯一写入者。
3. 客户端只提交意图和 `expected_version`，不提交完整状态、不自行生成事件或胜负。
4. 所有客户端可见对象均由服务端生成 Public/Viewer/Private DTO；API 不直接返回数据库 Entity。
5. REST 负责查询、幂等命令和长任务控制；WebSocket 负责已提交事件的实时投影、消息和状态补偿。
6. AI 生成、AI DM、NPC 对话和复盘只能提交候选、文本或意图；正式状态仍由 Go 校验并落库。

### 0.2 与数据库的命名映射

| API 概念 | 数据库/存储 |
| --- | --- |
| `script` | `scripts` |
| `script_version` | `script_versions` |
| `generation_job` | `generation_jobs` |
| `room` | `game_rooms` |
| `room_event` | `room_events` |
| `game` | `game_sessions` |
| `game_event` | `game_events` |
| `game_projection` | `game_sessions + game_states + game_roles + 当前 viewer ACL` |
| `message` | `game_messages` |
| `vote` | `game_votes` |
| `ai_run` | `mv_ai.agent_runs` |

API 不暴露 `secret_payload_id`、密文、KMS、Redis key、内部 Prompt、完整 Agent trace 或数据库表名。

### 0.3 版本策略

- URL 主版本固定在路径：`/api/v1`。
- 同一 Major 内只允许新增可选字段、新增事件类型、新增错误码和向后兼容枚举。
- 删除字段、改变语义、改变默认行为必须发布 `/api/v2`。
- WebSocket 使用子协议 `mvws.v1`；协议不兼容变更需要 `mvws.v2`。
- OpenAPI `info.version` 使用 SemVer，例如 `1.3.0`；路径主版本仍为 `v1`。
- 客户端发送 `X-Client-Version`，服务端对过旧客户端阻止不兼容写操作。

---

## 1. 通用协议规范

### 1.1 HTTP 基础

| 项目 | 规范 |
| --- | --- |
| 协议 | HTTPS only；生产 HTTP 重定向 HTTPS |
| Content-Type | `application/json; charset=utf-8` |
| 文件上传 | 先申请预签名 URL，浏览器直传对象存储 |
| 字符编码 | UTF-8；JSON 使用 snake_case |
| 时间 | UTC RFC 3339，毫秒精度可选 |
| ID | 纯 26 字符 ULID 字符串；不暴露自增 ID，不使用资源类型前缀 |
| 请求追踪 | 客户端可带 `X-Request-Id`，否则服务端生成 |
| 响应缓存 | 公共 D0 可缓存；D1-D4 默认 `Cache-Control: no-store` |
| 条件更新 | 编辑器使用 `If-Match: "<row_version>"`；游戏使用 body `expected_version` |

### 1.2 公共请求头

| Header | 必需 | 用途 |
| --- | --- | --- |
| `Cookie: mv_session=...` | 登录接口外必需 | 服务端 Session |
| `X-CSRF-Token` | 所有非 GET/HEAD 请求必需 | CSRF 防护 |
| `Idempotency-Key` | 所有会产生持久状态变化的非 GET 请求必需 | 幂等 |
| `X-Request-Id` | 建议 | 链路追踪 |
| `X-Client-Version` | 建议 | 协议兼容 |
| `X-Client-Platform` | 可选 | `web`、`admin`、测试端 |
| `Origin` | 浏览器请求必需 | 服务端校验允许来源 |
| `If-Match` | 编辑器条件更新建议 | 行版本条件 |

### 1.3 成功响应信封

单对象：

```json
{
  "data": {
    "id": "01J9Z3K4M7Q2R8T5V6W1X30000"
  },
  "meta": {
    "request_id": "01J9Z3K4M7Q2R8T5V6W1XZ0000",
    "server_time": "2026-11-20T20:00:00Z",
    "api_version": "v1"
  }
}
```

分页列表：

```json
{
  "data": {
    "items": [],
    "next_cursor": "01J9Z3K4M7Q2R8T5V6W1X50000"
  },
  "meta": {
    "request_id": "01J9Z3K4M7Q2R8T5V6W1XZ0000",
    "server_time": "2026-11-20T20:00:00Z",
    "api_version": "v1"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `data` | object | 当前接口业务 DTO |
| `meta.request_id` | string | 请求追踪 |
| `meta.server_time` | string | 服务端时间 |
| `meta.api_version` | string | `v1` |
| `meta.idempotent_replay` | boolean | 可选；重复幂等请求返回原结果时为 `true` |
| `data.next_cursor` | string/null | 游标分页下一页 |
| `data.items` | array | 当前页 |

### 1.4 错误响应信封

```json
{
  "error": {
    "code": "STATE_VERSION_CONFLICT",
    "message": "游戏状态已更新，请刷新后重试",
    "details": {
      "expected_version": 17,
      "current_version": 18
    },
    "retryable": true,
    "request_id": "01J9Z3K4M7Q2R8T5V6W1XZ0000",
    "trace_id": "01J9Z3K4M7Q2R8T5V6W1X20000"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `error.code` | string | 稳定机器错误码 |
| `error.message` | string | 用户/开发者可读摘要，不包含内部堆栈 |
| `error.details` | object/null | 字段错误、版本、当前状态等安全明细 |
| `error.retryable` | boolean | 客户端是否可用相同幂等键重试 |
| `error.request_id` | string | 请求追踪 |
| `error.trace_id` | string | 内部 trace 定位 |

### 1.5 参数和枚举

路径参数使用 `snake_case` 占位：`{script_id}`、`{game_id}`、`{object_type}`。

公共分页参数：

| 参数 | 类型 | 约束 |
| --- | --- | --- |
| `cursor` | string | 不透明游标；不得解码后在客户端拼接 |
| `limit` | integer | 默认 20，最大 100 |
| `sort` | string | 白名单字段 |

公共写入参数：

| 参数 | 类型 | 约束 |
| --- | --- | --- |
| `idempotency_key` | header string | 1-128 字符，同一 actor/scope 唯一 |
| `expected_version` | integer | 游戏状态写操作必需 |
| `client_request_id` | string | 可选；用于客户端诊断和 WebSocket 对账 |

示例中的 `...` 仅用于缩短 ULID 展示，实际请求和响应必须使用完整 26 字符 ID；`<64-hex>` 是安全占位符。密码字段中的 `<plaintext-over-tls>` 只表示 TLS 请求体中的值，不得作为真实测试或文档密钥提交。

---

## 2. 认证、授权与安全

### 2.1 MVP 认证方案

MVP 采用服务端 Session，不使用长期浏览器 JWT：

1. 登录成功后服务端创建 `user_sessions`。
2. 浏览器保存 `HttpOnly + Secure + SameSite=Lax` 的 `mv_session` Cookie。
3. 非 GET 请求必须同时发送 `X-CSRF-Token`，服务端校验 Session、CSRF、Origin。
4. WebSocket 先调用 `POST /api/v1/realtime/tickets` 获取一次性 `ws_ticket`。
5. `ws_ticket` 绑定 `user_id + room_id + purpose + nonce`，短有效期、单次使用。
6. 断开登录态后，旧 Cookie 和旧 WebSocket ticket 失效；已建立连接也必须停止写操作。

未来原生端或第三方集成可以增加 JWT Provider，但必须复用同一授权服务和短期 WebSocket ticket 模型，不得让 JWT 绕过房间/角色权限。

当前 Task 5 实现基线：

- MVP 首个 Provider 固定为 `email`；API 接收邮件地址和密码，不实现验证码或第三方 OAuth。
- 密码仅在 TLS 请求体中传输，服务端使用 Argon2id PHC 哈希保存，禁止记录原文。
- 注册、登录成功返回一次性 CSRF Token；浏览器 Session Token 仅写入 `HttpOnly` Cookie，API 和日志均不回显 Cookie 值。
- 幂等重放注册不会保存或复用原始 Session Token；服务端返回同一用户并签发新的短期 Session。
- 非 GET 请求必须带允许的 `Origin`；已登录写请求还必须带 `X-CSRF-Token`。

### 2.2 授权链

```text
Session
→ Account status
→ Content visibility
→ Room membership
→ Seat / role binding
→ Object ACL
→ Field ACL
→ Reveal gate
→ Action permission
→ Projection
```

接口权限等级：

| 等级 | 定义 |
| --- | --- |
| `anonymous` | 游客可访问公开内容 |
| `session` | 需要有效登录 Session |
| `session_csrf` | 需要登录 Session、CSRF 和 Origin |
| `room_member` | 当前有效房间成员 |
| `room_player` | 有有效玩家席位 |
| `game_player` | 当前 game 的有效玩家，且角色绑定有效 |
| `owner` | 创作者或资源所有者 |
| `host` | 房间当前房主 |
| `internal` | Go/Python 内部服务身份，不暴露公网 |

`host` 不自动获得玩家角色秘密、私聊或身份表权限。

`IDEMPOTENT_REPLAY` 不是错误码；幂等成功回放通过 `meta.idempotent_replay=true`
表达。

### 2.3 安全约束

- API 不直接返回数据库 Entity，只返回 DTO。
- 私密字段由服务端投影；前端隐藏、频道名和 Query Key 不是权限。
- `script_characters.private_payload`、`script_clues.private_payload`、`game_roles` 身份、私聊正文和 NPC 记忆不在公共 DTO。
- 狼人杀阵营信息只返回给规则允许的本人/阵营频道。
- 对象存储 URI 必须通过短期签名或受控下载接口获取。
- 请求体、查询参数、上传文件和 WebSocket 消息全部做 Schema、长度和频率校验。
- 所有内部 AI 调用必须携带 `actor_scope`、`game_id`、`version`、`trace_id` 和 `idempotency_key`。
- 不把完整 Prompt、Python 堆栈、SQL、KMS 密钥、数据库连接符返回客户端。

### 2.3.1 内部 Agent Runtime 执行协议

Task 8 增加内部端点：

```text
POST /internal/v1/ai/agent-runs
```

该端点不暴露给浏览器。请求与响应使用 Contracts `AgentRequest` /
`AgentResponse`。请求头必须包含 `X-Service-Id`、`X-Service-Token`、
`X-Service-Scope`、`X-Request-Id`、`X-Trace-Id` 和 `Idempotency-Key`。

生产安全 profile 要求 mTLS + `mv-hmac-sha256-v1`。配置签名密钥后还必须携带
`X-Request-Signature-Version`、`X-Service-Key-Id`、`X-Request-Timestamp`、
`X-Request-Nonce` 和 `X-Request-Signature`。HMAC 覆盖 signature version、
Key ID、timestamp、nonce、Service ID、method、path、query 和 body hash。
Key ID 支持 Active/Previous 轮换和撤销；生产 nonce 使用 Redis 原子消费，
Store/KMS 故障 fail-closed。

响应状态为 `succeeded|failed|timeout|cancelled`。`structured_output` 只能是
Python 已注册任务 Schema 的校验结果；Python 不返回 Go 状态补丁、事件或胜负。
工具调用记录只返回参数 hash、工具名、状态和耗时，不返回原文参数或模型隐藏推理。
响应包含 Schema key/version 和不含原始输入的结构化 `validation_errors`。
详细的内部网络、超时、取消、幂等和防循环约束见 `docs/api/ai-internal-api.md`。

DQ-18/DQ-19 已批准并落地 Schema Registry 与内部安全基础；Tool trust/外部 MCP
仍由 DQ-20 阻塞。生产 CA、KMS 和 Redis 多实例部署仍为 `NOT VERIFIED`。
批准与实施边界见 `docs/architecture/task-11-approval-blockers.md`。

### 2.4 限流建议

| 范围 | 建议 |
| --- | --- |
| 登录/验证码 | 按 IP + 账号 + 设备哈希，严格限流 |
| Chat/AI DM/NPC | 按 user + room + minute 限流 |
| 生成任务 | 按 user + project + day 配额 |
| 游戏意图 | 按 game + player + second 短时突发限制 |
| WebSocket 连接 | 按 user + IP + 房间连接数限制 |
| 公开搜索 | 按 IP/user + minute |

限流错误返回 `429 RATE_LIMITED`，包含安全的 `retry_after_seconds`。

---

## 3. REST API 完整清单

> 完整清单中的每一个 operation 都在后续章节有独立契约。未列出的内部 Go/Python API 不属于浏览器公共契约。

Task 5 已实现的浏览器 operation：

- 用户：U01、U02、U03、U04、U05。
- 房间：R01、R02、R03、R04、R05、R06、R10 的 `close/transfer_host/kick`、R12、R13、R14。
- 私人/邀请房创建时返回一次性邀请 token，供 R04 `invitation_token` 使用。

本任务未实现 U06、R07、R08、R09、R11 以及真实 WebSocket `room_chat` 广播；这些 operation 在相应依赖和门禁通过前不得由前端假定可用。

Task 6 已实现的游戏 operation（候选实现，受 Proposed ADR-004/005 门禁）：

- 开局：R09。
- 游戏查询：G01、G02、G06。
- 暂停/恢复：G09、G10。
- 剧本杀：M01、M02、M03、M04、M05、M07、M08。

Task 6 未实现 G03、G04、G05、G07、G08、真实 WebSocket game stream 和 AI DM。
剧本必须提供可编译的 `rule_templates.config_json.murder` 策略；阶段进入/退出条件 DSL
尚未冻结，本轮遇到非空条件对象会拒绝开局，不能用隐式默认规则补猜。

当前剧本杀运行状态机为：

```text
starting → running → paused → running
running → settling → completed
running/paused/starting → aborted
```

阶段按 `script_phases.sequence_no` 单调推进；读取完成、搜证配额用尽、投票完成或服务端
deadline 可触发推进。客户端不能提交阶段序号、事件、身份、票数或胜负。

### 3.1 用户与社交

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| U01 | POST | `/api/v1/auth/register` | 注册 | `anonymous` |
| U02 | POST | `/api/v1/auth/login` | 登录 | `anonymous` |
| U03 | POST | `/api/v1/auth/logout` | 退出当前设备 | `session_csrf` |
| U04 | GET | `/api/v1/me` | 获取本人摘要、资料、隐私和统计 | `session` |
| U05 | PATCH | `/api/v1/me` | 修改资料和隐私设置 | `session_csrf` |
| U06 | GET | `/api/v1/users/{user_id}/public-profile` | 查看公开主页 | `anonymous/session` |
| U07 | GET | `/api/v1/friends` | 好友和待处理申请 | `session` |
| U08 | POST | `/api/v1/friend-requests` | 发送好友申请 | `session_csrf` |
| U09 | PATCH | `/api/v1/friendships/{friendship_id}` | 接受、拒绝或删除好友 | `session_csrf` |
| U10 | POST | `/api/v1/users/{user_id}/block` | 拉黑用户 | `session_csrf` |

### 3.2 剧本与工坊

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| S01 | POST | `/api/v1/scripts` | 创建剧本项目 | `session_csrf` |
| S02 | GET | `/api/v1/scripts` | 公开剧本搜索/筛选 | `anonymous/session` |
| S03 | GET | `/api/v1/scripts/{script_id}` | 获取剧本详情或作者视图 | `anonymous/session/owner` |
| S04 | PATCH | `/api/v1/scripts/{script_id}` | 修改剧本基础信息 | `owner` |
| S05 | POST | `/api/v1/scripts/{script_id}/documents/upload-intents` | 获取原始文档上传凭证 | `owner` |
| S06 | POST | `/api/v1/script-documents/{document_id}/complete` | 确认文档上传 | `owner` |
| S07 | POST | `/api/v1/scripts/{script_id}/generation-jobs` | 启动解析/结构化/游戏生成 | `owner` |
| S08 | GET | `/api/v1/generation-jobs/{job_id}` | 查询生成任务状态 | `owner/auditor` |
| S09 | GET | `/api/v1/generation-jobs/{job_id}/events` | SSE 订阅任务进度 | `owner/auditor` |
| S10 | POST | `/api/v1/generation-jobs/{job_id}/actions` | 重试、取消或恢复人工审核 | `owner` |
| S11 | GET | `/api/v1/scripts/{script_id}/versions` | 版本列表 | `owner/reviewer` |
| S12 | POST | `/api/v1/scripts/{script_id}/versions/{version_id}/drafts` | 克隆草稿 | `owner` |
| S13 | GET | `/api/v1/scripts/{script_id}/versions/{version_id}/editor` | 编辑器结构化内容 | `owner/reviewer` |
| S14 | POST | `/api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}` | 新建结构化对象 | `owner` |
| S15 | PATCH | `/api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}` | 修改结构化对象 | `owner` |
| S16 | POST | `/api/v1/scripts/{script_id}/versions/{version_id}/regenerations` | 局部重生成 | `owner` |
| S17 | POST | `/api/v1/scripts/{script_id}/versions/{version_id}/validate` | 运行发布前校验 | `owner` |
| S18 | POST | `/api/v1/scripts/{script_id}/versions/{version_id}/publish` | 发布版本 | `owner` |
| S19 | POST | `/api/v1/publications/{publication_id}/unpublish` | 下架发布 | `owner/auditor` |
| S20 | GET | `/api/v1/scripts/{script_id}/versions/{version_id}/role-preview` | 角色视角预览 | `owner/reviewer` |
| S21 | DELETE | `/api/v1/scripts/{script_id}` | 软删除未发布剧本 | `owner` |
| S22 | DELETE | `/api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}` | 删除草稿结构化对象 | `owner` |

### 3.3 房间与实时入口

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| R01 | POST | `/api/v1/rooms` | 创建房间 | `session_csrf` |
| R02 | GET | `/api/v1/rooms` | 查询公开/可加入房间 | `anonymous/session` |
| R03 | GET | `/api/v1/rooms/{room_id}` | 获取房间当前投影 | `room_member/session` |
| R04 | POST | `/api/v1/rooms/{room_id}/join` | 加入房间 | `session_csrf` |
| R05 | POST | `/api/v1/rooms/{room_id}/leave` | 离开房间 | `room_member` |
| R06 | POST | `/api/v1/rooms/{room_id}/ready` | 设置准备状态 | `room_player` |
| R07 | POST | `/api/v1/rooms/{room_id}/invitations` | 创建邀请 | `room_member` |
| R08 | POST | `/api/v1/invitations/resolve` | 解析邀请并获取加入目标 | `session` |
| R09 | POST | `/api/v1/rooms/{room_id}/start` | 开局 | `host` |
| R10 | POST | `/api/v1/rooms/{room_id}/host-actions` | 房主控制动作 | `host` |
| R11 | POST | `/api/v1/realtime/tickets` | 获取一次性 WebSocket ticket | `room_member` |
| R12 | GET | `/api/v1/rooms/{room_id}/events` | 房间事件补偿 | `room_member` |
| R13 | GET | `/api/v1/rooms/{room_id}/messages` | 等待房聊天历史 | `room_member` |
| R14 | POST | `/api/v1/rooms/{room_id}/messages` | 发送等待房消息 | `room_member` |

### 3.4 游戏通用

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| G01 | GET | `/api/v1/games/{game_id}/projection` | 获取当前游戏投影 | `game_player` |
| G02 | GET | `/api/v1/games/{game_id}/events` | 查询投影后的事件增量 | `game_player` |
| G03 | POST | `/api/v1/games/{game_id}/reconnect` | 断线重连同步 | `game_player` |
| G04 | GET | `/api/v1/games/{game_id}/messages` | 查询聊天消息 | `game_player` |
| G05 | POST | `/api/v1/games/{game_id}/messages` | 发送聊天消息 | `game_player` |
| G06 | GET | `/api/v1/games/{game_id}/result` | 获取结算结果 | `game_player` |
| G07 | GET | `/api/v1/games/{game_id}/replay` | 获取授权复盘 | `game_player` |
| G08 | POST | `/api/v1/games/{game_id}/dm-turns` | AI DM 回合 | `game_player` |
| G09 | POST | `/api/v1/games/{game_id}/pause` | 暂停游戏 | `host/game_player` |
| G10 | POST | `/api/v1/games/{game_id}/resume` | 恢复游戏 | `host/game_player` |

### 3.5 剧本杀

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| M01 | GET | `/api/v1/games/{game_id}/murder/me` | 我的角色与阶段状态 | `game_player` |
| M02 | POST | `/api/v1/games/{game_id}/murder/reading-complete` | 标记角色阅读完成 | `game_player` |
| M03 | GET | `/api/v1/games/{game_id}/murder/search-points` | 可搜证点位 | `game_player` |
| M04 | POST | `/api/v1/games/{game_id}/murder/searches` | 执行搜证 | `game_player` |
| M05 | GET | `/api/v1/games/{game_id}/murder/clues` | 查看已解锁线索 | `game_player` |
| M06 | POST | `/api/v1/games/{game_id}/murder/questions` | 质询 | `game_player` |
| M07 | POST | `/api/v1/games/{game_id}/murder/votes` | 提交/修改投票 | `game_player` |
| M08 | GET | `/api/v1/games/{game_id}/murder/votes` | 查看规则允许的票型 | `game_player` |

### 3.6 狼人杀

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| W01 | GET | `/api/v1/games/{game_id}/werewolf/me` | 我的身份与技能视图 | `game_player` |
| W02 | GET | `/api/v1/games/{game_id}/werewolf/night-context` | 当前夜晚可行动作 | `game_player` |
| W03 | POST | `/api/v1/games/{game_id}/werewolf/night-actions` | 提交夜间行动 | `game_player` |
| W04 | POST | `/api/v1/games/{game_id}/werewolf/speeches` | 提交白天发言 | `game_player` |
| W05 | POST | `/api/v1/games/{game_id}/werewolf/votes` | 提交/修改投票 | `game_player` |
| W06 | POST | `/api/v1/games/{game_id}/werewolf/last-words` | 提交遗言 | `game_player` |
| W07 | POST | `/api/v1/games/{game_id}/werewolf/skills` | 触发被动/主动技能 | `game_player` |

### 3.7 RPG

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| P01 | GET | `/api/v1/games/{game_id}/rpg/context` | 当前场景、地图和合法动作 | `game_player` |
| P02 | POST | `/api/v1/games/{game_id}/rpg/moves` | 移动 | `game_player` |
| P03 | POST | `/api/v1/games/{game_id}/rpg/explores` | 探索 | `game_player` |
| P04 | POST | `/api/v1/games/{game_id}/rpg/actions/parse` | 自由文本解析 | `game_player` |
| P05 | POST | `/api/v1/games/{game_id}/rpg/actions` | 执行结构化动作 | `game_player` |
| P06 | POST | `/api/v1/games/{game_id}/rpg/npcs/{npc_id}/dialogues` | NPC 对话 | `game_player` |
| P07 | GET | `/api/v1/games/{game_id}/rpg/npcs/{npc_id}/dialogues` | NPC 对话历史 | `game_player` |
| P08 | GET | `/api/v1/games/{game_id}/rpg/tasks` | 当前任务 | `game_player` |
| P09 | POST | `/api/v1/games/{game_id}/rpg/tasks/{task_id}/actions` | 接受/放弃任务 | `game_player` |
| P10 | POST | `/api/v1/games/{game_id}/rpg/items/use` | 使用物品 | `game_player` |
| P11 | POST | `/api/v1/games/{game_id}/rpg/story-choices` | 提交剧情选择 | `game_player` |

### 3.8 AI 与社区

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| A01 | GET | `/api/v1/ai-runs/{ai_run_id}` | 查询脱敏 AI Run 状态 | `owner/game_player/auditor` |
| C01 | GET | `/api/v1/scripts/{script_id}/reviews` | 评分/短评列表 | `anonymous/session` |
| C02 | POST | `/api/v1/scripts/{script_id}/reviews` | 提交评分/短评 | `session_csrf/player_completed` |
| C03 | GET | `/api/v1/scripts/{script_id}/comments` | 评论列表 | `anonymous/session` |
| C04 | POST | `/api/v1/scripts/{script_id}/comments` | 发表评论 | `session_csrf` |
| C05 | PUT | `/api/v1/scripts/{script_id}/favorite` | 设置/取消收藏 | `session_csrf` |
| C06 | POST | `/api/v1/share-links` | 创建受控分享链接 | `session_csrf` |
| C07 | POST | `/api/v1/reports` | 举报内容/消息/玩家 | `session_csrf` |

### 3.9 运营后台

后台 API 与玩家 API 分离，完整契约见 `docs/api/admin-api.md`。

| ID | 方法 | URL | 名称 | 鉴权 |
| --- | --- | --- | --- | --- |
| AD01 | GET | `/api/v1/admin/scripts/review-queue` | 内容审核队列 | `content_reviewer` |
| AD02 | POST | `/api/v1/admin/scripts/{script_id}/versions/{version_id}/moderate` | 审核内容 | `content_reviewer` |
| AD03 | POST | `/api/v1/admin/scripts/{script_id}/versions/{version_id}/unpublish` | 下架内容 | `content_reviewer/auditor` |
| AD04 | GET | `/api/v1/admin/reports` | 举报列表 | `support_agent` |
| AD05 | POST | `/api/v1/admin/reports/{report_id}/actions` | 举报处置 | `support_agent` |
| AD06 | POST | `/api/v1/admin/users/{user_id}/sanctions` | 禁言/封禁 | `support_agent/security_auditor` |
| AD07 | GET | `/api/v1/admin/incidents` | 异常记录查询 | `support_agent/security_auditor` |
| AD08 | GET | `/api/v1/admin/generation/review-queue` | AI 内容审核队列 | `content_reviewer` |
| AD09 | GET | `/api/v1/admin/audit-logs` | 审计日志 | `security_auditor` |

---

## 4. 用户、剧本与工坊接口契约

### 4.1 用户与社交

#### U01 注册

- 方法与 URL：`POST /api/v1/auth/register`
- 鉴权/权限：`anonymous`；按 IP、设备哈希和账号标识限流。
- 请求字段：`provider`、`identifier`、`password`、`nickname`、`terms_version`、`privacy_version`。
- 请求示例：`{"provider":"email","identifier":"alice@example.com","password":"<plaintext-over-tls>","nickname":"Alice","terms_version":"2026-11","privacy_version":"2026-11"}`
- 响应结构：`user`、`profile`、`session_expires_at`、`csrf_token`。
- 响应示例：`{"data":{"user":{"id":"01J9Z3K4M7Q2R8T5V6W1XA0000","status":"active"},"profile":{"nickname":"Alice"},"session_expires_at":"2026-11-27T20:00:00Z","csrf_token":"csrf_xxx"}}`
- 权限规则：注册成功后仅返回本人最小资料；不得返回认证标识原文。
- 错误码：`VALIDATION_ERROR`、`VERIFICATION_CODE_INVALID`、`IDENTIFIER_ALREADY_EXISTS`、`RATE_LIMITED`、`TERMS_NOT_ACCEPTED`。
- 幂等性：`Idempotency-Key` 必需；同键同请求返回同一用户和 Session；不同 payload 返回 `IDEMPOTENCY_CONFLICT`。

#### U02 登录

- 方法与 URL：`POST /api/v1/auth/login`
- 鉴权/权限：`anonymous`；按 IP + identifier + device hash 限流。
- 请求字段：`provider`、`identifier`、`password`、`remember_device`。
- 请求示例：`{"provider":"email","identifier":"alice@example.com","password":"<plaintext-over-tls>","remember_device":true}`
- 响应结构：`user`、`profile_summary`、`session_expires_at`、`csrf_token`。
- 响应示例：`{"data":{"user":{"id":"01J9Z3K4M7Q2R8T5V6W1XA0000","status":"active"},"profile_summary":{"nickname":"Alice","avatar_url":null},"session_expires_at":"2026-11-27T20:00:00Z","csrf_token":"csrf_xxx"}}`
- 权限规则：账号 `suspended/disabled/deleted` 不能登录；错误消息不得泄露账号是否存在。
- 错误码：`AUTH_INVALID`、`ACCOUNT_RESTRICTED`、`VERIFICATION_CODE_EXPIRED`、`RATE_LIMITED`、`DEPENDENCY_ERROR`。
- 幂等性：可选 `Idempotency-Key`；重复登录可以创建新设备 Session，不视为业务写幂等。

#### U03 退出当前设备

- 方法与 URL：`POST /api/v1/auth/logout`
- 鉴权/权限：`session_csrf`。
- 请求字段：无；可带 `all_devices:false`。
- 请求示例：`{"all_devices":false}`
- 响应结构：`revoked_session_count`。
- 响应示例：`{"data":{"revoked_session_count":1}}`
- 权限规则：只能撤销当前用户会话；退出后旧 WebSocket 不可继续写。
- 错误码：`AUTH_REQUIRED`、`CSRF_INVALID`、`INTERNAL_ERROR`。
- 幂等性：`Idempotency-Key` 必需；重复退出返回 `revoked_session_count=0`。

#### U04 获取本人摘要

- 方法与 URL：`GET /api/v1/me`
- 鉴权/权限：`session`。
- 请求字段：Query `include=profile,privacy,statistics,recent_games`，逗号分隔。
- 请求示例：`GET /api/v1/me?include=profile,privacy,statistics`
- 响应结构：`user`、`profile`、`privacy`、`statistics`、`capabilities`。
- 响应示例：`{"data":{"user":{"id":"01J9Z3K4M7Q2R8T5V6W1XA0000","public_no":"MV000001","status":"active"},"profile":{"nickname":"Alice","locale":"zh-CN"},"privacy":{"history_visibility":"friends"},"statistics":{"games_completed":12,"games_won":5},"capabilities":["create_script","join_room"]}}`
- 权限规则：仅本人；隐私设置由服务端读取，不返回 D2-D4 内容。
- 错误码：`AUTH_REQUIRED`、`ACCOUNT_RESTRICTED`、`VALIDATION_ERROR`。
- 幂等性：GET 安全可重试，不要求幂等键。

#### U05 修改资料和隐私

- 方法与 URL：`PATCH /api/v1/me`
- 鉴权/权限：`session_csrf`；昵称/头像修改进入审核。
- 请求字段：`profile_patch`、`privacy_patch`、`row_version`。
- 请求示例：`{"row_version":3,"profile_patch":{"nickname":"Alice2","bio":"推理爱好者"},"privacy_patch":{"history_visibility":"private","online_visibility":"friends"}}`
- 响应结构：`profile`、`privacy`、`moderation_status`、`row_version`。
- 响应示例：`{"data":{"profile":{"nickname":"Alice2","bio":"推理爱好者"},"privacy":{"history_visibility":"private","online_visibility":"friends"},"moderation_status":"pending","row_version":4}}`
- 权限规则：只能修改本人资料；隐私枚举白名单；不允许修改 `public_no`、`user_id`、账号状态。
- 错误码：`VALIDATION_ERROR`、`NICKNAME_CONFLICT`、`VERSION_CONFLICT`、`MODERATION_REJECTED`。
- 幂等性：`Idempotency-Key` 必需；同键同 patch 返回首次结果。

#### U06 查看公开主页

- 方法与 URL：`GET /api/v1/users/{user_id}/public-profile`
- 鉴权/权限：`anonymous/session`；按用户隐私设置投影。
- 请求字段：Path `user_id`；Query `include=stats,scripts`。
- 请求示例：`GET /api/v1/users/01J9Z3K4M7Q2R8T5V6W1XA0000/public-profile?include=stats`
- 响应结构：`user_public_no`、`nickname`、`avatar_url`、`bio`、`stats_visibility`、可选 `statistics`、`public_scripts`。
- 响应示例：`{"data":{"user_public_no":"MV000001","nickname":"Alice","avatar_url":null,"bio":"推理爱好者","stats_visibility":"private","statistics":null,"public_scripts":[]}}`
- 权限规则：私密历史、在线状态、草稿和未发布内容永不返回；被拉黑或隐私限制时返回最小公开摘要。
- 错误码：`USER_NOT_FOUND`、`FORBIDDEN_VIEW`、`RATE_LIMITED`。
- 幂等性：GET 安全可重试。

#### U07 好友与待处理申请列表

- 方法与 URL：`GET /api/v1/friends`
- 鉴权/权限：`session`。
- 请求字段：`cursor`、`limit`、`include_pending`。
- 请求示例：`GET /api/v1/friends?include_pending=true&limit=30`
- 响应结构：分页 `friends`、`incoming_requests`、`outgoing_requests`、`blocked_count`。
- 响应示例：`{"data":{"items":[{"friendship_id":"01J9Z3K4M7Q2R8T5V6W1XFD0WS","user":{"id":"01J9Z3K4M7Q2R8T5V6W1XB0000","nickname":"Bob"},"status":"accepted","note":"固定车队"}],"incoming_requests":[{"friendship_id":"01J9Z3K4M7Q2R8T5V6W1XZH6V2","from_user":{"id":"01J9Z3K4M7Q2R8T5V6W1XC0000","nickname":"Carol"},"requested_at":"2026-11-20T19:00:00Z"}],"outgoing_requests":[]}}`
- 权限规则：只能读取与本人有关的友好关系；备注只归本人。
- 错误码：`AUTH_REQUIRED`、`VALIDATION_ERROR`。
- 幂等性：GET 安全可重试。

#### U08 发送好友申请

- 方法与 URL：`POST /api/v1/friend-requests`
- 鉴权/权限：`session_csrf`。
- 请求字段：`target_user_id`、`message`。
- 请求示例：`{"target_user_id":"01J9Z3K4M7Q2R8T5V6W1XB0000","message":"一起玩剧本杀"}`
- 响应结构：`friendship_id`、`status`、`requested_at`。
- 响应示例：`{"data":{"friendship_id":"01J9Z3K4M7Q2R8T5V6W1X20000","status":"pending","requested_at":"2026-11-20T20:00:00Z"}}`
- 权限规则：不能申请自己；拉黑、隐私拒绝、已有 pending/accepted 关系按规则返回冲突或现有关系。
- 错误码：`SELF_ACTION_FORBIDDEN`、`FRIENDSHIP_EXISTS`、`USER_BLOCKED`、`FRIEND_REQUEST_FORBIDDEN`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一 friendship。

#### U09 接受、拒绝或删除好友

- 方法与 URL：`PATCH /api/v1/friendships/{friendship_id}`
- 鉴权/权限：`session_csrf`；只有该好友关系参与方可操作。
- 请求字段：`action`：`accept|reject|remove`、`expected_status`。
- 请求示例：`{"action":"accept","expected_status":"pending"}`
- 响应结构：`friendship_id`、`status`、`responded_at`。
- 响应示例：`{"data":{"friendship_id":"01J9Z3K4M7Q2R8T5V6W1X20000","status":"accepted","responded_at":"2026-11-20T20:01:00Z"}}`
- 权限规则：不能由非参与方操作；被拉黑时按安全规则返回拒绝。
- 错误码：`FORBIDDEN_ACTION`、`FRIENDSHIP_NOT_FOUND`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复执行同一 action 返回当前结果。

#### U10 拉黑用户

- 方法与 URL：`POST /api/v1/users/{user_id}/block`
- 鉴权/权限：`session_csrf`。
- 请求字段：`reason_code`、`note`。
- 请求示例：`{"reason_code":"harassment","note":"持续骚扰"}`
- 响应结构：`blocked_user_id`、`blocked_at`。
- 响应示例：`{"data":{"blocked_user_id":"01J9Z3K4M7Q2R8T5V6W1XB0000","blocked_at":"2026-11-20T20:02:00Z"}}`
- 权限规则：不能拉黑自己；拉黑影响好友申请、邀请和私聊；房间内保留必要公共交互。
- 错误码：`SELF_ACTION_FORBIDDEN`、`USER_NOT_FOUND`、`VALIDATION_ERROR`。
- 幂等性：`Idempotency-Key` 必需；重复拉黑返回现有关系。

### 4.2 剧本、生成任务和编辑

#### S01 创建剧本项目

- 方法与 URL：`POST /api/v1/scripts`
- 鉴权/权限：`session_csrf`；用户需具备创作能力且未超配额。
- 请求字段：`title`、`game_type`、`rights_type`、`language`、`target_players`、`duration_minutes`、`difficulty`、`content_rating`（默认 `all`）、`rule_template_id`（可选；只允许引用平台受控 active 模板）。
- 请求示例：`{"title":"雾港最后一班船","game_type":"script_murder","rights_type":"original","language":"zh-CN","target_players":{"min":4,"max":8},"duration_minutes":90,"difficulty":"standard","content_rating":"16+","rule_template_id":"01J9Z3K4M7Q2R8T5V6W1XR0000"}`
- 响应结构：`script_id`、`status`、`owner_user_id`、`active_draft_version_id`。
- 响应示例：`{"data":{"script_id":"01J9Z3K4M7Q2R8T5V6W1X30000","status":"draft","owner_user_id":"01J9Z3K4M7Q2R8T5V6W1XA0000","active_draft_version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000"}}`
- 权限规则：创建者成为 owner；`private_trial` 不代表公开传播授权。
- 错误码：`VALIDATION_ERROR`、`QUOTA_EXCEEDED`、`ACCOUNT_RESTRICTED`、`RIGHTS_DECLARATION_REQUIRED`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一 `script`，不重复创建版本。

#### S02 搜索和筛选公开剧本

- 方法与 URL：`GET /api/v1/scripts`
- 鉴权/权限：`anonymous/session`；只索引公开/授权可见内容。
- 请求字段：`q`、`game_type`、`min_players`、`max_players`、`duration_max`、`difficulty`、`tags`、`sort`、`cursor`、`limit`。
- 请求示例：`GET /api/v1/scripts?q=悬疑&game_type=script_murder&min_players=6&sort=popular&limit=20`
- 响应结构：分页 `script_card`，字段 `script_id`、`public_version_id`、`title`、`cover_url`、`game_type`、`players`、`duration_minutes`、`difficulty`、`rating`、`tags`、`availability`。
- 响应示例：`{"data":{"items":[{"script_id":"01J9Z3K4M7Q2R8T5V6W1X30000","public_version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000","title":"雾港最后一班船","game_type":"script_murder","players":{"min":4,"max":8},"duration_minutes":90,"difficulty":"standard","rating":4.6,"tags":["悬疑","社会派"],"availability":"available"}],"next_cursor":null}}`
- 权限规则：私人草稿、下架正文、未公开角色和真相不参与索引。
- 错误码：`VALIDATION_ERROR`、`FORBIDDEN_VIEW`、`RATE_LIMITED`。
- 幂等性：GET 安全可重试。

#### S03 获取剧本详情

- 方法与 URL：`GET /api/v1/scripts/{script_id}`
- 鉴权/权限：`anonymous/session` 读取公开版本；`owner` 可请求编辑视图。
- 请求字段：Path `script_id`；Query `version_id` 可选、`view=public|editor`。
- 请求示例：`GET /api/v1/scripts/01J9Z3K4M7Q2R8T5V6W1X30000?version_id=01J9Z3K4M7Q2R8T5V6W1XW0000&view=public`
- 响应结构：公共 DTO：元数据、无剧透简介、标签、版本摘要、评分摘要；作者 DTO额外 `validation_status`、`current_draft`。
- 响应示例：`{"data":{"script_id":"01J9Z3K4M7Q2R8T5V6W1X30000","title":"雾港最后一班船","summary":"一艘没有归期的船。","game_type":"script_murder","players":{"min":4,"max":8},"duration_minutes":90,"difficulty":"standard","public_version":{"id":"01J9Z3K4M7Q2R8T5V6W1XW0000","version_no":3,"published_at":"2026-11-20T18:00:00Z"},"rating":{"score":4.6,"count":28},"viewer_capabilities":["create_room","favorite"]}}`
- 权限规则：公开视图不含私密角色本、真相和搜索点隐藏信息；作者编辑视图需 owner。
- Task 7 实现说明：匿名 S03 只返回当前公开版本；作者使用受保护的 S13 获取 Draft/Editor 视图。
- 错误码：`CONTENT_NOT_FOUND`、`FORBIDDEN_VIEW`、`CONTENT_UNAVAILABLE`。
- 幂等性：GET 安全可重试。

#### S04 修改剧本基础信息

- 方法与 URL：`PATCH /api/v1/scripts/{script_id}`
- 鉴权/权限：`owner`；`session_csrf`。
- 请求字段：`row_version`、`patch`，允许 `title`、`summary`、`cover_asset_uri`、`language`、`content_rating`、`min_players`、`max_players`、`duration_minutes`、`difficulty`。
- 请求示例：`{"row_version":4,"patch":{"summary":"封闭船舱中的最后一夜","difficulty":"advanced","tags":["悬疑","封闭空间"]}}`
- 响应结构：更新后的 metadata、`row_version`、`moderation_status`。
- 响应示例：`{"data":{"script_id":"01J9Z3K4M7Q2R8T5V6W1X30000","summary":"封闭船舱中的最后一夜","difficulty":"advanced","tags":["悬疑","封闭空间"],"row_version":5,"moderation_status":"pending"}}`
- 权限规则：不能修改 owner、发布版本指针和已发布版本正文。
- 错误码：`FORBIDDEN_EDIT`、`VERSION_CONFLICT`、`VALIDATION_ERROR`、`MODERATION_REJECTED`。
- 幂等性：`Idempotency-Key` 必需；同键返回首次更新结果。

#### S05 获取原始文档上传凭证

- 方法与 URL：`POST /api/v1/scripts/{script_id}/documents/upload-intents`
- 鉴权/权限：`owner`；校验配额、权利声明和允许格式。
- 请求字段：`filename`、`mime_type`、`byte_size`、`sha256`、`purpose=source_document`。
- 请求示例：`{"filename":"剧本.docx","mime_type":"application/vnd.openxmlformats-officedocument.wordprocessingml.document","byte_size":1048576,"sha256":"<64-hex>","purpose":"source_document"}`
- 响应结构：`document_id`、`upload_url`、`headers`、`expires_at`、`max_bytes`。
- 响应示例：`{"data":{"document_id":"01J9Z3K4M7Q2R8T5V6W1X5M4F1","upload_url":"https://storage.example/...","headers":{"Content-Type":"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},"expires_at":"2026-11-20T20:10:00Z","max_bytes":52428800}}`
- 权限规则：上传 URL 只能写指定隔离区对象；不能给公开 Bucket 或任意路径。
- 错误码：`UNSUPPORTED_FILE_TYPE`、`FILE_TOO_LARGE`、`QUOTA_EXCEEDED`、`RIGHTS_DECLARATION_REQUIRED`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一 document/upload intent。

#### S06 确认文档上传

- 方法与 URL：`POST /api/v1/script-documents/{document_id}/complete`
- 鉴权/权限：`owner`；只允许确认自己项目的 document。
- 请求字段：`sha256`、`byte_size`、`client_uploaded_at`。
- 请求示例：`{"sha256":"<64-hex>","byte_size":1048576,"client_uploaded_at":"2026-11-20T20:03:00Z"}`
- 响应结构：`document`、`scan_status`、`parse_status`、`detected_mime`。
- 响应示例：`{"data":{"document":{"id":"01J9Z3K4M7Q2R8T5V6W1X5M4F1","mime_type":"...docx","page_count":null},"scan_status":"pending","parse_status":"uploaded","detected_mime":"..."}}`
- 权限规则：服务端重新校验对象存在、大小、哈希和 MIME；失败不进入解析。
- 错误码：`DOCUMENT_NOT_FOUND`、`UPLOAD_NOT_FOUND`、`FILE_HASH_MISMATCH`、`FILE_REJECTED`。
- 幂等性：`Idempotency-Key` 必需；重复确认返回同一扫描状态。

#### S07 启动解析/生成任务

- 方法与 URL：`POST /api/v1/scripts/{script_id}/generation-jobs`
- 鉴权/权限：`owner`；文档必须 clean 且配额可用。
- 请求字段：`job_type`、`source_document_id`、`target_mode`、`workflow_config`、`budget`。
- 请求示例：`{"job_type":"generate","source_document_id":"01J9Z3K4M7Q2R8T5V6W1X5M4F1","target_mode":"script_murder","workflow_config":{"players":{"min":6,"max":8},"duration_minutes":90,"difficulty":"standard","allow_ai_expansion":true},"budget":{"max_cost":20.0,"max_repair_rounds":3}}`
- 响应结构：`job_id`、`status`、`progress_bp`、`created_at`、`links.events`。
- 响应示例：`{"data":{"job_id":"01J9Z3K4M7Q2R8T5V6W1XM0000","status":"queued","progress_bp":0,"created_at":"2026-11-20T20:04:00Z","links":{"events":"/api/v1/generation-jobs/01J9Z3K4M7Q2R8T5V6W1XM0000/events"}}}`
- 权限规则：job 只能写项目 staging；不能直接发布或修改已发布版本。
- 错误码：`DOCUMENT_NOT_READY`、`GENERATION_QUOTA_EXCEEDED`、`SCHEMA_INVALID`、`DEPENDENCY_ERROR`。
- 幂等性：`Idempotency-Key` 必需；唯一键为 project + job_type + key。

#### S08 查询生成任务状态

- 方法与 URL：`GET /api/v1/generation-jobs/{job_id}`
- 鉴权/权限：`owner/reviewer/auditor`。
- 请求字段：Path `job_id`。
- 请求示例：`GET /api/v1/generation-jobs/01J9Z3K4M7Q2R8T5V6W1XM0000`
- 响应结构：`job_id`、`job_type`、`status`、`progress_bp`、`current_step`、`steps`、`validation`、`error`、`artifact_summary`。
- 响应示例：`{"data":{"job_id":"01J9Z3K4M7Q2R8T5V6W1XM0000","job_type":"generate","status":"running","progress_bp":4200,"current_step":"clue","steps":[{"step_key":"parse","status":"succeeded","progress_bp":10000},{"step_key":"clue","status":"running","progress_bp":5000}],"validation":{"blocking_count":0},"error":null,"artifact_summary":{"characters":6,"clues":18}}}`
- 权限规则：进度只返回脱敏摘要，不返回完整 Prompt、原文片段和私密生成内容。
- 错误码：`JOB_NOT_FOUND`、`FORBIDDEN_VIEW`、`VALIDATION_ERROR`。
- 幂等性：GET 安全可重试。

#### S09 订阅生成任务进度

- 方法与 URL：`GET /api/v1/generation-jobs/{job_id}/events`
- 鉴权/权限：`owner/reviewer/auditor`；`Accept: text/event-stream`。
- 请求字段：Path `job_id`；Query `last_event_id` 可选。
- 请求示例：`GET /api/v1/generation-jobs/01J9Z3K4M7Q2R8T5V6W1XM0000/events?last_event_id=evt_123`
- 响应结构：SSE `id` + `event` + JSON data，事件类型为 `generation.progress`、`generation.step_changed`、`generation.failed`、`generation.succeeded`。
- 响应示例：`event: generation.progress\ndata: {"job_id":"01J9Z3K4M7Q2R8T5V6W1XM0000","progress_bp":4200,"current_step":"clue","status":"running"}`
- 权限规则：只允许 owner/reviewer；事件已脱敏，不包含完整模型输入输出。
- 错误码：`JOB_NOT_FOUND`、`FORBIDDEN_VIEW`、`STREAM_UNAVAILABLE`。
- 幂等性：SSE 可重连；以 `id` 去重，不应重复产生任务副作用。

#### S10 生成任务控制

- 方法与 URL：`POST /api/v1/generation-jobs/{job_id}/actions`
- 鉴权/权限：`owner`。
- 请求字段：`action`：`retry|cancel|resume_human_review`、`stage_key`、`review_decision`、`comment`。
- 请求示例：`{"action":"retry","stage_key":"clue","comment":"修复线索可达性"}`
- 响应结构：`job_id`、`status`、`accepted_action`、`next_poll_after_ms`。
- 响应示例：`{"data":{"job_id":"01J9Z3K4M7Q2R8T5V6W1XM0000","status":"queued","accepted_action":"retry","next_poll_after_ms":1000}}`
- 权限规则：仅对本人任务；已取消/已发布/已拒绝状态按规则返回非法状态。
- 错误码：`JOB_INVALID_STATE`、`FORBIDDEN_ACTION`、`IDEMPOTENCY_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同一 action 重试返回首次 decision。

#### S11 版本列表

- 方法与 URL：`GET /api/v1/scripts/{script_id}/versions`
- 鉴权/权限：`owner/reviewer`；公共版本通过 S03 返回。
- 请求字段：`status`、`limit`；当前响应保留 `cursor` 位置，但 Task 7 使用版本号倒序分页。
- 请求示例：`GET /api/v1/scripts/01J9Z3K4M7Q2R8T5V6W1X30000/versions?status=draft&limit=20`
- 响应结构：分页 `version_id`、`version_no`、`status`、`schema_version`、`validation_status`、`blocking_issue_count`、`row_version`、`created_at`、`published_at`。
- 响应示例：`{"data":{"items":[{"version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000","version_no":3,"status":"draft","schema_version":"1.0.0","validation_status":"pending","created_at":"2026-11-20T20:00:00Z","published_at":null}],"next_cursor":null}}`
- 权限规则：作者可看草稿；审核员只看审核相关状态；其他用户不可列草稿。
- 错误码：`FORBIDDEN_VIEW`、`SCRIPT_NOT_FOUND`、`VALIDATION_ERROR`。
- 幂等性：GET 安全可重试。

#### S12 克隆草稿

- 方法与 URL：`POST /api/v1/scripts/{script_id}/versions/{version_id}/drafts`
- 鉴权/权限：`owner`；源版本可读且未处于不可编辑状态。
- 请求字段：`draft_name`、`rule_template_id`（可选）；`base_version_id` 由 Path 的 `version_id` 明确。
- 请求示例：`{"draft_name":"v4-线索修订","rule_template_id":"01J9Z3K4M7Q2R8T5V6W1XR0000"}`
- 响应结构：`new_version_id`、`version_no`、`status=draft`、`row_version`。
- 响应示例：`{"data":{"new_version_id":"01J9Z3K4M7Q2R8T5V6W1XX0000","version_no":4,"status":"draft","row_version":1}}`
- 权限规则：草稿归属当前 owner；不得修改源发布版本。
- 错误码：`VERSION_NOT_FOUND`、`FORBIDDEN_EDIT`、`VERSION_FROZEN`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一草稿。

#### S13 获取编辑器内容

- 方法与 URL：`GET /api/v1/scripts/{script_id}/versions/{version_id}/editor`
- 鉴权/权限：`owner/reviewer`。
- 请求字段：Query `include=story_facts,characters,relationships,scenes,phases,clues,search_points,tasks,events,world_settings,npcs`。
- 请求示例：`GET /api/v1/scripts/01J9Z3K4M7Q2R8T5V6W1X30000/versions/01J9Z3K4M7Q2R8T5V6W1XW0000/editor?include=characters,clues,phases`
- 响应结构：`script`、`version`、`chapters`、`stages`、`characters`、`clues`、`endings`、`assets`、`search_points`、`point_clues`。私密正文只返回授权引用，不返回密文。
- 响应示例：`{"data":{"version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000","row_version":7,"schema_version":"1.0.0","object_tree":{"characters":6,"clues":18,"phases":5},"validation":{"blocking_count":0,"warning_count":2},"objects":{"characters":[{"id":"01J9Z3K4M7Q2R8T5V6W1XD0000","display_name":"林岚","locked_fields":["private_goal"]}]}}}`
- 权限规则：作者编辑视图可含私密内容，但必须按角色编辑权限投影；审核员只读。
- 错误码：`VERSION_NOT_FOUND`、`FORBIDDEN_VIEW`、`SCHEMA_UNSUPPORTED`。
- 幂等性：GET 安全可重试。

#### S14 新建结构化对象

- 方法与 URL：`POST /api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}`
- 鉴权/权限：`owner`；当前版本必须为 `draft`。
- 请求字段：Path `object_type` 白名单、`object`、`expected_row_version`。Task 7 白名单为 `chapter|stage|character|clue|ending|asset|search_point|search_point_clue`。
- 请求示例：`{"object":{"character_key":"lin_lan","display_name":"林岚","public_profile":"船长"},"expected_row_version":7,"lock_fields":["display_name"]}`
- 响应结构：`object_id`、`object_type`、`row_version`、`issues`。
- 响应示例：`{"data":{"object_id":"01J9Z3K4M7Q2R8T5V6W1XD0000","object_type":"character","row_version":8,"issues":[]}}`
- 权限规则：不能创建未知类型；字段必须通过该对象 Schema。
- 错误码：`UNSUPPORTED_OBJECT_TYPE`、`SCHEMA_INVALID`、`VERSION_FROZEN`、`VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键同对象返回同一 object id。

#### S15 修改结构化对象

- 方法与 URL：`PATCH /api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}`
- 鉴权/权限：`owner`；草稿可编辑且对象未锁定。
- 请求字段：`object`（完整替换对象）、`expected_row_version`。版本级 `row_version` 在每次对象写入后递增。
- 请求示例：`{"patch":{"public_profile":"雾港船长","public_goals":["寻找失踪船员"]},"expected_row_version":8,"lock_fields":["public_profile"]}`
- 响应结构：更新后对象摘要、`row_version`、`validation_delta`。
- 响应示例：`{"data":{"object_id":"01J9Z3K4M7Q2R8T5V6W1XD0000","row_version":9,"validation_delta":{"blocking_count":0,"warning_count":1}}}`
- 权限规则：不能更新锁字段、已发布版本、其他作者项目；对象类型与 ID 必须匹配。
- 错误码：`OBJECT_NOT_FOUND`、`OBJECT_TYPE_MISMATCH`、`LOCKED_FIELD`、`VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复同 patch 返回当前 row_version。

#### S16 局部重生成

- 方法与 URL：`POST /api/v1/scripts/{script_id}/versions/{version_id}/regenerations`
- 鉴权/权限：`owner`；目标草稿可编辑。
- 请求字段：`target_object_type`、`target_object_id`、`problem_ids`、`locked_fields`、`instruction`、`budget`。
- 请求示例：`{"target_object_type":"clue","target_object_id":"01J9Z3K4M7Q2R8T5V6W1XG0000","problem_ids":["v_clue_unreachable"],"locked_fields":["clue_key"],"instruction":"让该线索在第二轮可被发现","budget":{"max_cost":2.0}}`
- 响应结构：`job_id`、`candidate_id`、`status`、`estimated_cost`。
- 响应示例：`{"data":{"job_id":"01J9Z3K4M7Q2R8T5V6W1XGR9Z0","candidate_id":null,"status":"queued","estimated_cost":1.2}}`
- 权限规则：只能修改目标对象和必要依赖；不能改锁定字段和 Canon。
- 错误码：`TARGET_NOT_FOUND`、`LOCKED_FIELD`、`QUOTA_EXCEEDED`、`WORKFLOW_FAILED`。
- 幂等性：`Idempotency-Key` 必需；同键不生成重复候选或重复扣费。

#### S17 运行发布前校验

- 方法与 URL：`POST /api/v1/scripts/{script_id}/versions/{version_id}/validate`
- 鉴权/权限：`owner`；草稿或审核版本。
- 请求字段：`expected_row_version`、`validation_profile`（`publish|structure`）。
- 请求示例：`{"expected_row_version":11,"validation_profile":"publish"}`
- 响应结构：`status`、`blocking_issue_count`、`warning_issue_count`、`info_issue_count`、`issues[]`；每个 issue 含 `error_code`、`severity`、`path`、`message`、`state`。未批准 DSL 返回 `state=unsupported`。
- 响应示例：`{"data":{"validation_id":"01J9Z3K4M7Q2R8T5V6W1XY44J2","status":"blocked","blocking_issues":[{"issue_id":"v1","object_type":"clue","object_id":"01J9Z3K4M7Q2R8T5V6W1XG0000","code":"CLUE_UNREACHABLE","message":"线索无法从任何搜证点获得"}],"warnings":[],"simulation_summary":{"completed":false},"report_url":null}}`
- 权限规则：作者可运行；校验结果只读，不自动清除阻断问题。
- 错误码：`SCHEMA_INVALID`、`VALIDATION_TIMEOUT`、`FORBIDDEN_EDIT`、`DEPENDENCY_ERROR`。
- 幂等性：`Idempotency-Key` 必需；同版本同校验配置可返回已存在结果。

#### S18 发布版本

- 方法与 URL：`POST /api/v1/scripts/{script_id}/versions/{version_id}/publish`
- 鉴权/权限：`owner`；版本未冻结，阻断问题为 0，审核状态满足渠道要求。
- 请求字段：`channel`：`private|invite|public`、`expected_row_version`、`content_rating_confirm`。发布事务内重新执行校验，不信任旧 `validation_id`。
- 请求示例：`{"channel":"private","expected_row_version":11,"content_rating_confirm":true}`
- 响应结构：`publication_id`、`status`、`published_version_id`、`published_at`、`moderation_status`。
- 响应示例：`{"data":{"publication_id":"01J9Z3K4M7Q2R8T5V6W1XPAM8C","status":"published","published_version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000","published_at":"2026-11-20T20:10:00Z","moderation_status":"not_required"}}`
- 权限规则：公开渠道必须审核；Task 7 在审核流程未批准/未实现前返回 `MODERATION_REQUIRED`。私有/邀请发布是原子切换，不能部分更新版本。
- 错误码：`PUBLISH_BLOCKED`、`VALIDATION_STALE`、`MODERATION_REQUIRED`、`VERSION_FROZEN`、`IDEMPOTENCY_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复发布返回同一 publication。

#### S19 下架发布

- 方法与 URL：`POST /api/v1/publications/{publication_id}/unpublish`
- 鉴权/权限：`owner/auditor`；需要 `session_csrf`。
- 请求字段：`reason_code`、`reason`、`expected_status`（MVP 仅接受 `active`）。
- 请求示例：`{"reason_code":"author_request","reason":"修正版权授权","expected_status":"active"}`
- 响应结构：`publication_id`、`status`、`unlisted_at`、`active_room_policy`。
- 响应示例：`{"data":{"publication_id":"01J9Z3K4M7Q2R8T5V6W1XPAM8C","status":"unlisted","unlisted_at":"2026-11-20T20:12:00Z","active_room_policy":"rooms_may_finish"}}`
- 权限规则：下架阻止新房间，不强制终止进行中房间；运营操作写审计。
- 错误码：`PUBLICATION_NOT_FOUND`、`FORBIDDEN_ACTION`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复下架返回当前状态。

#### S20 角色视角预览

- 方法与 URL：`GET /api/v1/scripts/{script_id}/versions/{version_id}/role-preview`
- 鉴权/权限：`owner/reviewer`。
- 请求字段：Query `role_id`、`phase_key`、`include_known_facts`。
- 请求示例：`GET /api/v1/scripts/01J9Z3K4M7Q2R8T5V6W1X30000/versions/01J9Z3K4M7Q2R8T5V6W1XW0000/role-preview?role_id=01J9Z3K4M7Q2R8T5V6W1XD0000&phase_key=search_round_1`
- 响应结构：按该角色视角裁剪的 `projected_content`、`visible_clues`、`known_relationships`、`available_actions`。
- 响应示例：`{"data":{"role_id":"01J9Z3K4M7Q2R8T5V6W1XD0000","projected_content":{"private_script":{"title":"林岚","body":"你在午夜听见..."},"known_relationships":[{"target":"01J9Z3K4M7Q2R8T5V6W1XE0000","label":"弟弟"}]},"visible_clues":["01J9Z3K4M7Q2R8T5V6W1XG0000"],"available_actions":["search","private_chat"]}}`
- 权限规则：仅作者/审核员；不能返回其他角色秘密、完整真相或数据库密文。
- 错误码：`ROLE_NOT_FOUND`、`PHASE_NOT_FOUND`、`FORBIDDEN_VIEW`。
- 幂等性：GET 安全可重试。

#### S21 软删除未发布剧本

- 方法与 URL：`DELETE /api/v1/scripts/{script_id}`
- 鉴权/权限：`owner`；`session_csrf`。
- 请求字段：`row_version`、`reason`。
- 权限规则：只允许软删除无有效发布指针的剧本；已发布剧本先执行 S19 下架。历史版本与引用对局不物理删除。
- 错误码：`SCRIPT_NOT_FOUND`、`FORBIDDEN_EDIT`、`VERSION_CONFLICT`、`VERSION_FROZEN`。
- 幂等性：`Idempotency-Key` 必需。

#### S22 删除草稿结构化对象

- 方法与 URL：`DELETE /api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}`
- 鉴权/权限：`owner`；目标版本必须为 `draft`。
- 请求字段：`expected_row_version`。
- 权限规则：发布版本不可删除对象；被其他对象引用的章节、阶段、线索或搜证点由数据库外键拒绝。
- 错误码：`OBJECT_NOT_FOUND`、`VERSION_FROZEN`、`VERSION_CONFLICT`、`UNSUPPORTED_OBJECT_TYPE`。
- 幂等性：`Idempotency-Key` 必需。

---

## 5. 房间与游戏通用接口契约

### 5.1 房间

当前 MVP 房间规则：

- 创建容量必须落在剧本版本允许的 `min_players..max_players` 内；空位不阻止等待，但不能满足开局条件。
- 同一用户最多一个有效席位；重复加入返回原席位，不重复占位。
- 最后席位由 `game_rooms` 行锁和 `room_members(room_id, active_seat_no)` 唯一约束共同保护，并发加入只允许一个成功。
- 房主离开等待房时，按加入时间转移给最早的仍有效成员；没有成员时房间转为 `cancelled`。
- 等待房默认 2 小时过期，可由 `AUTH_ROOM_WAITING_TTL` 调整；Worker 关闭过期房间并追加 `room.closed` 事件。
- 开局后不能通过普通 join/leave/ready/room chat 接口继续操作等待房。
- 等待房消息默认上限 4000 字符，按 `user + room` 每分钟 10 条限流；`chat_seq` 独立于 `room_events.seq` 和 `game_events.seq`。

#### R01 创建房间

- 方法与 URL：`POST /api/v1/rooms`
- 鉴权/权限：`session_csrf`；用户可创建房间且内容版本可用。
- 请求字段：`script_version_id`、`game_type`、`rule_template_id`、`title`、`visibility`、`password`、`capacity`、`mode_config`、`privacy_confirmed`。
- 请求示例：`{"script_version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000","game_type":"script_murder","title":"周五晚雾港局","visibility":"password","password":"4826","capacity":{"min":6,"max":6},"mode_config":{"search_rounds":2,"allow_private_chat":true},"privacy_confirmed":true}`
- 响应结构：`room`、`member`、`invite_hint`、`room_event_seq`。
- 响应示例：`{"data":{"room":{"id":"01J9Z3K4M7Q2R8T5V6W1X20000","status":"waiting","game_type":"script_murder","visibility":"password","capacity":{"min":6,"max":6},"host_user_id":"01J9Z3K4M7Q2R8T5V6W1XA0000"},"member":{"id":"01J9Z3K4M7Q2R8T5V6W1XR0000","seat_no":1,"room_role":"host","member_status":"joined"},"invite_hint":{"code_enabled":true},"room_event_seq":1}}`
- 权限规则：只能使用可发布/私有可玩的内容版本；密码只返回是否设置，不返回明文。
- 错误码：`VALIDATION_ERROR`、`CONTENT_UNAVAILABLE`、`PLAYER_COUNT_MISMATCH`、`QUOTA_EXCEEDED`、`IDEMPOTENCY_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一房间和房主席位。

#### R02 查询公开/可加入房间

- 方法与 URL：`GET /api/v1/rooms`
- 鉴权/权限：`anonymous/session`；私人房间不返回。
- 请求字段：`game_type`、`status=waiting`、`joinable`、`script_id`、`cursor`、`limit`。
- 请求示例：`GET /api/v1/rooms?game_type=script_murder&joinable=true&limit=20`
- 响应结构：分页 `room_card`：`room_id`、`game_type`、`script_title`、`host_public_name`、`players`、`capacity`、`visibility`、`has_password`、`status`。
- 响应示例：`{"data":{"items":[{"room_id":"01J9Z3K4M7Q2R8T5V6W1X20000","game_type":"script_murder","script_title":"雾港最后一班船","host_public_name":"Alice","players":3,"capacity":6,"visibility":"public","has_password":false,"status":"waiting"}],"next_cursor":null}}`
- 权限规则：仅公开房间；不得通过列表暴露私人房间码、密码或成员私密资料。
- 错误码：`VALIDATION_ERROR`、`RATE_LIMITED`。
- 幂等性：GET 安全可重试。

#### R03 获取房间投影

- 方法与 URL：`GET /api/v1/rooms/{room_id}`
- 鉴权/权限：`room_member/session`。
- 请求字段：Path `room_id`；Query `view=waiting|host`。
- 请求示例：`GET /api/v1/rooms/01J9Z3K4M7Q2R8T5V6W1X20000?view=waiting`
- 响应结构：`room`、`members`、`seats`、`viewer_capabilities`、`room_event_seq`、`game_summary`。
- 响应示例：`{"data":{"room":{"id":"01J9Z3K4M7Q2R8T5V6W1X20000","status":"waiting","capacity":{"min":6,"max":6}},"members":[{"user":{"id":"01J9Z3K4M7Q2R8T5V6W1XA0000","nickname":"Alice"},"seat_no":1,"room_role":"host","member_status":"joined"}],"viewer_capabilities":["ready","start_game"],"room_event_seq":7,"game_summary":null}}`
- 权限规则：房间成员可看等待信息；房主看不到其他人的角色私密信息；“可开始”只是投影，最终由 start 服务端校验。
- 错误码：`ROOM_NOT_FOUND`、`NOT_ROOM_MEMBER`、`ROOM_CLOSED`、`FORBIDDEN_VIEW`。
- 幂等性：GET 安全可重试。

#### R04 加入房间

- 方法与 URL：`POST /api/v1/rooms/{room_id}/join`
- 鉴权/权限：`session_csrf`；可访问房间且未被拉黑。
- 请求字段：`room_code`、`invitation_token`、`password`、`preferred_seat_no`。
- 请求示例：`{"invitation_token":"01J9Z3K4M7Q2R8T5V6W1X3WG88","password":"4826","preferred_seat_no":4}`
- 响应结构：`member`、`room_projection`、`room_event_seq`、`game_id` 可选。
- 响应示例：`{"data":{"member":{"id":"01J9Z3K4M7Q2R8T5V6W1XS0000","seat_no":4,"room_role":"player","member_status":"joined"},"room_projection":{"id":"01J9Z3K4M7Q2R8T5V6W1X20000","status":"waiting"},"room_event_seq":8,"game_id":null}}`
- 权限规则：服务端校验邀请、密码、黑名单、人数和席位；一个用户一个有效席位。
- 错误码：`ROOM_NOT_FOUND`、`ROOM_FULL`、`INVITATION_INVALID`、`PASSWORD_INVALID`、`USER_BLOCKED`、`ALREADY_JOINED`、`ROOM_NOT_WAITING`。
- 幂等性：`Idempotency-Key` 必需；重复加入返回原席位，不会重复占位。

#### R05 离开房间

- 方法与 URL：`POST /api/v1/rooms/{room_id}/leave`
- 鉴权/权限：`room_member`；等待阶段普通成员可退出。
- 请求字段：`reason`、`expected_room_event_seq`。
- 请求示例：`{"reason":"user_cancel","expected_room_event_seq":8}`
- 响应结构：`member_status`、`room_status`、`host_transfer_required`。
- 响应示例：`{"data":{"member_status":"left","room_status":"waiting","host_transfer_required":false}}`
- 权限规则：开局后不能自由退出，只能申请托管/放弃；房主退出触发迁移规则。
- 错误码：`ROOM_NOT_FOUND`、`NOT_ROOM_MEMBER`、`ROOM_IN_GAME`、`HOST_TRANSFER_REQUIRED`。
- 幂等性：`Idempotency-Key` 必需；已离开成员重复请求返回当前 left 状态。

#### R06 设置准备状态

- 方法与 URL：`POST /api/v1/rooms/{room_id}/ready`
- 鉴权/权限：`room_player`；等待阶段。
- 请求字段：`ready`、`expected_room_event_seq`。
- 请求示例：`{"ready":true,"expected_room_event_seq":9}`
- 响应结构：`member`、`ready_count`、`can_start`、`room_event_seq`。
- 响应示例：`{"data":{"member":{"id":"01J9Z3K4M7Q2R8T5V6W1XS0000","member_status":"ready"},"ready_count":6,"can_start":true,"room_event_seq":10}}`
- 权限规则：只能修改本人准备状态；房主准备不影响其他席位。
- 错误码：`ROOM_NOT_WAITING`、`NOT_ROOM_MEMBER`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复设置同值返回当前状态。

#### R07 创建邀请

- 方法与 URL：`POST /api/v1/rooms/{room_id}/invitations`
- 鉴权/权限：`room_member`；等待阶段。
- 请求字段：`max_uses`、`expires_in_seconds`、`invitee_user_ids` 可选。
- 请求示例：`{"max_uses":10,"expires_in_seconds":3600,"invitee_user_ids":["01J9Z3K4M7Q2R8T5V6W1XB0000"]}`
- 响应结构：`invitation_token`、`share_url`、`expires_at`、`max_uses`。
- 响应示例：`{"data":{"invitation_token":"01J9Z3K4M7Q2R8T5V6W1X3WG88","share_url":"https://mysteryverse.example/invite/01J9Z3K4M7Q2R8T5V6W1X3WG88","expires_at":"2026-11-20T21:00:00Z","max_uses":10}}`
- 权限规则：邀请不绕过密码、黑名单、人数和登录要求；token 只存哈希。
- 错误码：`ROOM_NOT_WAITING`、`NOT_ROOM_MEMBER`、`INVITATION_CREATE_FORBIDDEN`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一 token，不重复增加使用额度。

#### R08 解析邀请

- 方法与 URL：`POST /api/v1/invitations/resolve`
- 鉴权/权限：`session`。
- 请求字段：`invitation_token`。
- 请求示例：`{"invitation_token":"01J9Z3K4M7Q2R8T5V6W1X3WG88"}`
- 响应结构：`room_id`、`game_type`、`script_summary`、`requires_password`、`joinable`。
- 响应示例：`{"data":{"room_id":"01J9Z3K4M7Q2R8T5V6W1X20000","game_type":"script_murder","script_summary":{"title":"雾港最后一班船","players":{"min":6,"max":6}},"requires_password":true,"joinable":true}}`
- 权限规则：解析不代表加入，不扩大内容或房间权限；无效/过期 token 不返回房间私密信息。
- 错误码：`INVITATION_INVALID`、`INVITATION_EXPIRED`、`ROOM_CLOSED`、`ROOM_FULL`。
- 幂等性：GET 语义，POST 建议带 `Idempotency-Key`；不产生状态变化。

#### R09 开始游戏

- 方法与 URL：`POST /api/v1/rooms/{room_id}/start`
- 鉴权/权限：`host`；全员准备、人数合法、版本/规则配置有效。
- 请求字段：`expected_room_event_seq`、`content_version_id`、`rules_ack`。
- 请求示例：`{"expected_room_event_seq":12,"content_version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000","rules_ack":true}`
- 响应结构：`game_id`、`room_status`、`state_version`、`game_event_seq`、`engine_version`、`game_type`。
- 响应示例：`{"data":{"game_id":"01J9Z3K4M7Q2R8T5V6W1X10000","room_status":"in_game","state_version":1,"game_event_seq":1,"engine_version":"murder-engine-1.0.0","game_type":"script_murder"}}`
- 权限规则：仅房主；开始请求必须原子创建 GameInstance 并锁定内容/规则版本；重复请求返回原 GameInstance。
- 错误码：`HOST_REQUIRED`、`ROOM_NOT_WAITING`、`PLAYERS_NOT_READY`、`PLAYER_COUNT_MISMATCH`、`CONTENT_VERSION_UNAVAILABLE`、`GAME_ALREADY_STARTED`。
- 幂等性：`Idempotency-Key` 必需；`room_id + idempotency_key` 唯一，重复返回同一 game。

#### R10 房主控制动作

- 方法与 URL：`POST /api/v1/rooms/{room_id}/host-actions`
- 鉴权/权限：`host`；不同 action 额外校验游戏阶段。
- 请求字段：`action`：`kick|transfer_host|close`、`target_member_id`、`reason`、`expected_room_event_seq`。
- 请求示例：`{"action":"kick","target_member_id":"01J9Z3K4M7Q2R8T5V6W1X30000","reason":"未准备","expected_room_event_seq":10}`
- 响应结构：`accepted_action`、`room_status`、`members`/`room_event_seq`、`requires_confirmation`。
- 响应示例：`{"data":{"accepted_action":"kick","room_status":"waiting","members":[{"id":"01J9Z3K4M7Q2R8T5V6W1X40000","room_role":"host"}],"room_event_seq":11,"requires_confirmation":false}}`
- 权限规则：`kick` 仅等待阶段；`transfer_host` 必须指定在线成员；`close` 不影响进行中对局结果的审计。游戏内 `pause/resume` 使用 G09/G10，不得经本接口执行。
- 错误码：`HOST_REQUIRED`、`TARGET_NOT_MEMBER`、`ROOM_IN_GAME`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同一 action 重试返回首次结果。

#### R11 获取 WebSocket ticket

- 方法与 URL：`POST /api/v1/realtime/tickets`
- 鉴权/权限：`room_member`；目标房间有效。
- 请求字段：`room_id`、`game_id` 可选、`last_room_seq`、`last_game_seq`、`protocol_version`。
- 请求示例：`{"room_id":"01J9Z3K4M7Q2R8T5V6W1X20000","game_id":"01J9Z3K4M7Q2R8T5V6W1X10000","last_room_seq":12,"last_game_seq":184,"protocol_version":"mvws.v1"}`
- 响应结构：`ticket`、`ws_url`、`expires_at`、`connection_id_hint`。
- 响应示例：`{"data":{"ticket":"01J9Z3K4M7Q2R8T5V6W1XY0000","ws_url":"wss://api.mysteryverse.example/ws/v1","expires_at":"2026-11-20T20:00:30Z","connection_id_hint":"01J9Z3K4M7Q2R8T5V6W1XJ0000"}}`
- 权限规则：ticket 绑定 user + room + purpose + nonce；只能用于一次握手；不写入 LocalStorage。
- 错误码：`NOT_ROOM_MEMBER`、`ROOM_CLOSED`、`GAME_NOT_FOUND`、`PROTOCOL_UNSUPPORTED`、`RATE_LIMITED`。
- 幂等性：GET 语义，POST 可带 `Idempotency-Key`；ticket 短时且单次使用，重试可生成新 ticket。

#### R12 房间事件补偿

- 方法与 URL：`GET /api/v1/rooms/{room_id}/events`
- 鉴权/权限：`room_member`。
- 请求字段：Query `after_seq`、`limit`、`until_seq`。
- 请求示例：`GET /api/v1/rooms/01J9Z3K4M7Q2R8T5V6W1X20000/events?after_seq=11&limit=100`
- 响应结构：`events`、`current_seq`、`has_more`；事件已按 viewer 投影。
- 响应示例：`{"data":{"events":[{"event_id":"01J9Z3K4M7Q2R8T5V6W1X10000","event_type":"room.player_joined","seq":12,"payload":{"member":{"id":"01J9Z3K4M7Q2R8T5V6W1X20000","nickname":"Dave"},"seat_no":4}}],"current_seq":12,"has_more":false}}`
- 权限规则：只返回 room scope 内可见事件；私密事件必须按 viewer 投影后返回，不能只是前端过滤。
- 错误码：`ROOM_NOT_FOUND`、`NOT_ROOM_MEMBER`、`INVALID_SEQ_RANGE`、`EVENT_GAP_TOO_LARGE`。
- 幂等性：GET 安全可重试。

#### R13 查询等待房聊天

- 方法与 URL：`GET /api/v1/rooms/{room_id}/messages`
- 鉴权/权限：`room_member`。
- 请求字段：Query `cursor`、`limit`。
- 请求示例：`GET /api/v1/rooms/01J9Z3K4M7Q2R8T5V6W1X80000/messages?limit=30`
- 响应结构：分页 `items`，每项包含 `message_id`、`sender`、`text`、`chat_seq`、`sent_at`、`moderation_status`。
- 响应示例：`{"data":{"items":[{"message_id":"01J9Z3K4M7Q2R8T5V6W1X90000","sender":{"user_id":"01J9Z3K4M7Q2R8T5V6W1X50000","display_name":"Alice"},"text":"大家准备好了吗？","chat_seq":18,"sent_at":"2026-11-21T10:00:00Z","moderation_status":"allowed"}],"next_cursor":null}}`
- 权限规则：只允许有效房间成员读取；聊天流与 `room_events.seq` 独立。
- 错误码：`ROOM_NOT_FOUND`、`NOT_ROOM_MEMBER`、`INVALID_CURSOR`。
- 幂等性：GET 安全可重试。

#### R14 发送等待房消息

- 方法与 URL：`POST /api/v1/rooms/{room_id}/messages`
- 鉴权/权限：`room_member`；房间未关闭且发送者未被禁言。
- 请求字段：`client_message_id`、`text`。
- 请求示例：`{"client_message_id":"01J9Z3K4M7Q2R8T5V6W1X10000","text":"大家准备好了吗？"}`
- 响应结构：`message`、`event_id`、`chat_seq`。
- 响应示例：`{"data":{"message":{"id":"01J9Z3K4M7Q2R8T5V6W1X90000","text":"大家准备好了吗？","sender":{"user_id":"01J9Z3K4M7Q2R8T5V6W1X50000"},"sent_at":"2026-11-21T10:00:00Z"},"event_id":"01J9Z3K4M7Q2R8T5V6W1XA0000","chat_seq":18}}`
- 权限规则：消息写入 `room_messages`；不得写 `game_events`，不得伪造 `game_id`。
- 错误码：`ROOM_NOT_FOUND`、`NOT_ROOM_MEMBER`、`ROOM_CLOSED`、`MUTED`、`MESSAGE_TOO_LONG`、`RATE_LIMITED`。
- 幂等性：`Idempotency-Key` 必需；`room_id + sender_user_id + client_message_id` 唯一。

### 5.2 游戏通用

#### G01 获取游戏投影

- 方法与 URL：`GET /api/v1/games/{game_id}/projection`
- 鉴权/权限：`game_player`；必须已加入当前 game。
- 请求字段：Path `game_id`；Query `include=state,players,actions,my_role`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/projection?include=state,players,actions`
- 响应结构：`game_id`、`game_type`、`state_version`、`event_seq`、`viewer`、`phase`、`players`、`public_state`、`private_state`、`available_actions`。
- 响应示例：`{"data":{"game_id":"01J9Z3K4M7Q2R8T5V6W1X10000","game_type":"script_murder","state_version":17,"event_seq":184,"viewer":{"player_id":"01J9Z3K4M7Q2R8T5V6W1X40000","role_id":"01J9Z3K4M7Q2R8T5V6W1XD0000"},"phase":{"key":"search_round_1","deadline_at":"2026-11-20T20:08:00Z"},"players":[{"player_id":"01J9Z3K4M7Q2R8T5V6W1X40000","seat_no":1,"status":"active"}],"public_state":{"progress":{"searched":2,"required":6}},"private_state":{"search_quota":1},"available_actions":["search","chat","dm_hint"]}}`
- 权限规则：只返回当前 viewer 可见字段；角色私密信息只对本人；房主不自动获得他人私密视图。
- 错误码：`GAME_NOT_FOUND`、`GAME_NOT_STARTED`、`NOT_GAME_PLAYER`、`GAME_ALREADY_ENDED`。
- 幂等性：GET 安全可重试。

#### G02 查询事件增量

- 方法与 URL：`GET /api/v1/games/{game_id}/events`
- 鉴权/权限：`game_player`；事件必须按当前 viewer 投影。
- 请求字段：Query `after_seq`、`limit`、`before_seq`、`include_private`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/events?after_seq=184&limit=100`
- 响应结构：`events`、`current_seq`、`state_version`、`has_more`、`requires_snapshot`。
- 响应示例：`{"data":{"events":[{"event_id":"01J9Z3K4M7Q2R8T5V6W1X30000","event_type":"game.clue_unlocked","seq":185,"payload":{"clue":{"id":"01J9Z3K4M7Q2R8T5V6W1X40000","title":"船票残片","visibility":"private"},"available_actions":["view_clue"]}}],"current_seq":185,"state_version":18,"has_more":false,"requires_snapshot":false}}`
- 权限规则：`include_private` 只允许本人/角色 scope；每个事件已经是投影结果，不直接序列化 `game_events.payload`。
- 错误码：`GAME_NOT_FOUND`、`NOT_GAME_PLAYER`、`INVALID_SEQ_RANGE`、`EVENT_GAP_TOO_LARGE`。
- 幂等性：GET 安全可重试。

#### G03 游戏断线重连

- 方法与 URL：`POST /api/v1/games/{game_id}/reconnect`
- 鉴权/权限：`game_player`；同时校验最近有效 Session 和房间席位。
- 请求字段：`last_room_seq`、`last_game_seq`、`last_version`、`protocol_version`、`request_snapshot`。
- 请求示例：`{"last_room_seq":12,"last_game_seq":184,"last_version":17,"protocol_version":"mvws.v1","request_snapshot":true}`
- 响应结构：`snapshot` 或 `delta`、`state_version`、`event_seq`、`room_seq`、`requires_full_resync`、`viewer`。
- 响应示例：`{"data":{"snapshot":{"state_version":18,"event_seq":185,"phase":"search_round_1","private_state":{"role_id":"01J9Z3K4M7Q2R8T5V6W1XD0000"},"available_actions":["search"]},"delta":null,"state_version":18,"event_seq":185,"room_seq":12,"requires_full_resync":false,"viewer":{"player_id":"01J9Z3K4M7Q2R8T5V6W1X40000"}}}`
- 权限规则：重连不等于恢复旧凭据；旧 Session 失效时拒绝；事件差量必须按 viewer 权限裁剪。
- 错误码：`AUTH_REQUIRED`、`GAME_NOT_FOUND`、`NOT_GAME_PLAYER`、`STATE_VERSION_STALE`、`EVENT_GAP_TOO_LARGE`。
- 幂等性：会话语义可重试；`Idempotency-Key` 建议携带；不改变游戏状态。

#### G04 查询游戏消息

- 方法与 URL：`GET /api/v1/games/{game_id}/messages`
- 鉴权/权限：`game_player`；按 scope 读取。
- 请求字段：`channel=public|private|system`、`peer_player_id` 可选、`after_message_id`、`cursor`、`limit`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/messages?channel=public&limit=30`
- 响应结构：分页 `message`：`message_id`、`sender`、`message_type`、`visibility`、`text`、`sent_at`、`moderation_status`。
- 响应示例：`{"data":{"items":[{"message_id":"01J9Z3K4M7Q2R8T5V6W1XN0000","sender":{"type":"user","player_id":"01J9Z3K4M7Q2R8T5V6W1X40000","display_name":"Alice"},"message_type":"public","visibility":"room","text":"我认为船票时间不对。","sent_at":"2026-11-20T20:04:00Z","moderation_status":"allowed"}],"next_cursor":null}}`
- 权限规则：私聊只返回参与双方或明确授权旁听者；不能通过 `peer_player_id` 读取他人私聊；房主/创作者无额外读取权。
- 错误码：`GAME_NOT_FOUND`、`NOT_GAME_PLAYER`、`FORBIDDEN_VIEW`、`INVALID_MESSAGE_RANGE`。
- 幂等性：GET 安全可重试。

#### G05 发送游戏消息

- 方法与 URL：`POST /api/v1/games/{game_id}/messages`
- 鉴权/权限：`game_player`；阶段允许聊天且未被禁言。
- 请求字段：`client_message_id`、`message_type=public|private`、`recipient_player_ids`、`text`、`expected_version`。
- 请求示例：`{"client_message_id":"01J9Z3K4M7Q2R8T5V6W1XH0000","message_type":"private","recipient_player_ids":["01J9Z3K4M7Q2R8T5V6W1X50000"],"text":"你昨晚在哪里？","expected_version":18}`
- 响应结构：`message`、`event_id`、`event_seq`、`state_version`。
- 响应示例：`{"data":{"message":{"id":"01J9Z3K4M7Q2R8T5V6W1XB0000","message_type":"private","sender":{"player_id":"01J9Z3K4M7Q2R8T5V6W1XC0000"},"recipient_player_ids":["01J9Z3K4M7Q2R8T5V6W1XD0000"],"text":"你昨晚在哪里？","sent_at":"2026-11-21T10:05:00Z"},"event_id":"01J9Z3K4M7Q2R8T5V6W1XE0000","event_seq":186,"state_version":18}}`
- 权限规则：服务端校验发言窗口、禁言、目标成员和可见范围；私聊不广播公共频道；客户端不得提交 `system` 或 `system_note`。消息递增 `event_seq`，不改变规则状态时不递增 `state_version`。
- 错误码：`PHASE_NOT_ALLOWED`、`MUTED`、`TARGET_NOT_IN_ROOM`、`MESSAGE_TOO_LONG`、`RATE_LIMITED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；`client_message_id` 在 room/sender 下唯一；重试返回原消息。

#### G06 获取结算结果

- 方法与 URL：`GET /api/v1/games/{game_id}/result`
- 鉴权/权限：`game_player`；完成/异常终止后可见。
- 请求字段：Query `view=public|participant`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/result?view=participant`
- 响应结构：`result_id`、`status`、`winner`、`result_code`、`revealed_roles`、`summary`、`result_hash`、`ai_review_status`。
- 响应示例：`{"data":{"result_id":"01J9Z3K4M7Q2R8T5V6W1XQ0000","status":"completed","winner":{"team":"innocents","label":"好人阵营"},"result_code":"murder_solved","revealed_roles":[{"player_id":"01J9Z3K4M7Q2R8T5V6W1X40000","role_name":"林岚","role_revealed":true}],"summary":{"votes":5,"target":"01J9Z3K4M7Q2R8T5V6W1X60000"},"result_hash":"sha256:...","ai_review_status":"pending"}}`
- 权限规则：规则结果优先展示；结算前不能返回真相；AI 点评失败不阻塞结果。
- 错误码：`RESULT_NOT_FOUND`、`GAME_NOT_STARTED`、`NOT_GAME_PLAYER`、`RESULT_INCOMPLETE`。
- 幂等性：GET 安全可重试。

#### G07 获取复盘

- 方法与 URL：`GET /api/v1/games/{game_id}/replay`
- 鉴权/权限：`game_player`；进行中默认不可用，异常事故可按授权审计。
- 请求字段：Query `viewer_role_id`、`phase_key`、`include_ai_annotations`、`cursor`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/replay?viewer_role_id=01J9Z3K4M7Q2R8T5V6W1XD0000&include_ai_annotations=true`
- 响应结构：`replay_report`、`timeline`、`viewer_scope`、`ai_annotation_status`、`share_card` 可选。
- 响应示例：`{"data":{"replay_report":{"id":"01J9Z3K4M7Q2R8T5V6W1X50000","status":"ready","timeline_url":null},"timeline":[{"seq":12,"event_type":"game.clue_unlocked","summary":"公开线索：甲板上的血迹","visibility":"role","viewer_visible":true}],"viewer_scope":{"role_id":"01J9Z3K4M7Q2R8T5V6W1X60000"},"ai_annotation_status":"ready","share_card":{"available":true}}}`
- 权限规则：时间线来自权威事件；角色视角切换重新鉴权；私聊和私人剧本不因结算自动公开。
- 错误码：`REPLAY_NOT_READY`、`FORBIDDEN_VIEW`、`REPLAY_VERSION_UNSUPPORTED`、`GAME_NOT_FOUND`。
- 幂等性：GET 安全可重试。

#### G08 AI DM 回合

- 方法与 URL：`POST /api/v1/games/{game_id}/dm-turns`
- 鉴权/权限：`game_player`；当前阶段允许提示/解释，且未超出提示配额。
- 请求字段：`kind=explain_rule|narrate|hint|summarize|clarify`、`text`、`phase_key`、`expected_version`、`idempotency_key`。
- 请求示例：`{"kind":"hint","text":"我们不知道下一步该搜哪里","phase_key":"search_round_1","expected_version":18}`
- 响应结构：`ai_run_id`、`status=ready|degraded`、`message`、`referenced_event_ids`、`safety_status`、`state_version`。
- 响应示例：`{"data":{"ai_run_id":"01J9Z3K4M7Q2R8T5V6W1XT0000","status":"ready","message":{"id":"01J9Z3K4M7Q2R8T5V6W1XQYBA1","text":"第二轮搜证可检查甲板与船员舱；我只能指向已公开线索。","speaker":"AI DM"},"referenced_event_ids":["evt_160","evt_172"],"safety_status":"passed","state_version":18}}`
- 权限规则：AI 只接收 `dm_allowed_knowledge`；模型不能推进阶段、执行动作或直接返回真相；超时返回模板。
- 错误码：`PHASE_NOT_ALLOWED`、`AI_TIMEOUT`、`AI_SAFETY_BLOCKED`、`PROVIDER_UNAVAILABLE`、`HINT_QUOTA_EXHAUSTED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键不重复扣提示次数或重复写 AI 消息事件。

#### G09 暂停游戏

- 方法与 URL：`POST /api/v1/games/{game_id}/pause`
- 鉴权/权限：`host/game_player`；模式允许暂停且当前状态可暂停。
- 请求字段：`expected_version`。
- 请求示例：`{"expected_version":12}`
- 响应结构：`game_id`、`status=paused`、`state_version`、`event_seq`、`remaining_ms`。
- 响应示例：`{"data":{"game_id":"01J9Z3K4M7Q2R8T5V6W1X10000","status":"paused","state_version":13,"event_seq":186,"remaining_ms":245000}}`
- 权限规则：独立 GameSession 控制命令；Go 校验房主、模式规则、状态和 timer，产生 `game.paused` 权威事件。
- 错误码：`HOST_REQUIRED`、`PAUSE_NOT_ALLOWED`、`STATE_VERSION_STALE`、`STATE_VERSION_CONFLICT`、`IDEMPOTENCY_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复同 payload 返回首次结果，不重复产生事件。

#### G10 恢复游戏

- 方法与 URL：`POST /api/v1/games/{game_id}/resume`
- 鉴权/权限：`host/game_player`；当前游戏状态为 `paused`。
- 请求字段：`expected_version`。
- 请求示例：`{"expected_version":13}`
- 响应结构：`game_id`、`status=running`、`state_version`、`event_seq`、`deadline_at`。
- 响应示例：`{"data":{"game_id":"01J9Z3K4M7Q2R8T5V6W1X10000","status":"running","state_version":14,"event_seq":187,"deadline_at":"2026-11-21T10:08:00Z"}}`
- 权限规则：Go 重新设置权威 deadline，产生 `game.resumed` 权威事件。
- 错误码：`HOST_REQUIRED`、`PAUSE_NOT_ALLOWED`、`STATE_VERSION_STALE`、`STATE_VERSION_CONFLICT`、`IDEMPOTENCY_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复同 payload 返回首次结果。

---

## 6. 三类游戏模式接口契约

### 6.1 剧本杀

#### M01 获取我的角色与阶段状态

- 方法与 URL：`GET /api/v1/games/{game_id}/murder/me`
- 鉴权/权限：`game_player`；角色已分配。
- 请求字段：Query `include=private_script,goals,known_relationships,search_quota,available_actions`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/murder/me?include=private_script,goals,search_quota,available_actions`
- 响应结构：`player_id`、`role`、`private_script`、`goals`、`search_quota`、`known_relationships`、`available_actions`。
- 响应示例：`{"data":{"player_id":"01J9Z3K4M7Q2R8T5V6W1X40000","role":{"role_id":"01J9Z3K4M7Q2R8T5V6W1XD0000","name":"林岚","faction":"civilian"},"private_script":{"body":"你在午夜听见甲板传来..."},"goals":["找出凶手","隐藏你偷走船票的事实"],"search_quota":{"used":1,"remaining":1},"known_relationships":[{"role_id":"01J9Z3K4M7Q2R8T5V6W1XE0000","label":"弟弟","known_facts":["他欠你钱"]}],"available_actions":["search","view_clues","question","chat","vote"]}}`
- 权限规则：仅本人角色；不得返回其他角色私人剧本、凶手真相或隐藏目标。
- 错误码：`GAME_NOT_FOUND`、`GAME_NOT_STARTED`、`ROLE_NOT_BOUND`、`FORBIDDEN_VIEW`。
- 幂等性：GET 安全可重试。

#### M02 标记角色阅读完成

- 方法与 URL：`POST /api/v1/games/{game_id}/murder/reading-complete`
- 鉴权/权限：`game_player`；角色阅读阶段。
- 请求字段：`expected_version`。
- 请求示例：`{"expected_version":3}`
- 响应结构：`completed`、`completed_count`、`required_count`、`state_version`、`available_actions`。
- 响应示例：`{"data":{"completed":true,"completed_count":5,"required_count":6,"state_version":4,"available_actions":["wait"]}}`
- 权限规则：只能标记本人；重复请求不改变他人完成状态。
- 错误码：`PHASE_NOT_ALLOWED`、`ROLE_NOT_BOUND`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复请求返回当前 completed 状态。

#### M03 获取可搜证点位

- 方法与 URL：`GET /api/v1/games/{game_id}/murder/search-points`
- 鉴权/权限：`game_player`；搜证阶段。
- 请求字段：Query `phase_key`、`include_available_clues`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/murder/search-points?phase_key=search_round_1&include_available_clues=false`
- 响应结构：`search_points`：`point_id`、`name`、`description`、`state`、`remaining_global_uses`、`can_search`。
- 响应示例：`{"data":{"search_points":[{"point_id":"01J9Z3K4M7Q2R8T5V6W1XV0000","name":"船员舱","description":"狭窄的上下铺与木箱","state":"available","remaining_global_uses":null,"can_search":true}],"viewer_remaining_searches":1}}`
- 权限规则：只返回当前阶段开放点位；隐藏点位和未揭示线索不返回；取得者身份按冲突策略隐藏。
- 错误码：`PHASE_NOT_ALLOWED`、`GAME_NOT_FOUND`、`FORBIDDEN_VIEW`。
- 幂等性：GET 安全可重试。

#### M04 执行搜证

- 方法与 URL：`POST /api/v1/games/{game_id}/murder/searches`
- 鉴权/权限：`game_player`；阶段、搜证次数、点位和角色权限均合法。
- 请求字段：`search_point_id`、`expected_version`、`client_request_id`。
- 请求示例：`{"search_point_id":"01J9Z3K4M7Q2R8T5V6W1XV0000","expected_version":17,"client_request_id":"search_01"}`
- 响应结构：`intent_id`、`status=executed|rejected`、`clue`（可按本人/公共投影）、`claimed`、`remaining_quota`、`state_version`、`event_range`。
- 响应示例：`{"data":{"intent_id":"01J9Z3K4M7Q2R8T5V6W1XK0000","status":"executed","clue":{"id":"01J9Z3K4M7Q2R8T5V6W1XG0000","title":"船票残片","visibility":"private","body":"票根上的时间被划去，只留下23:40。"},"claimed":{"claim_slot":1,"exclusive":true},"remaining_quota":0,"state_version":18,"event_range":{"start_seq":185,"end_seq":186}}}`
- 权限规则：独占线索原子争抢；失败者不扣次数；线索按 visibility 投影；不返回取得者身份当规则要求隐藏。
- 错误码：`PHASE_NOT_ALLOWED`、`SEARCH_QUOTA_EXHAUSTED`、`SEARCH_POINT_UNAVAILABLE`、`CLUE_ALREADY_CLAIMED`、`STATE_VERSION_CONFLICT`、`IDEMPOTENCY_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键不重复扣次数；不同 payload 同键冲突。

#### M05 查看已解锁线索

- 方法与 URL：`GET /api/v1/games/{game_id}/murder/clues`
- 鉴权/权限：`game_player`；按当前阶段和角色 ACL 返回。
- 请求字段：Query `scope=public|private|mine|all_visible`、`phase_key`、`cursor`、`limit`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/murder/clues?scope=all_visible&limit=50`
- 响应结构：分页 `clue`：`clue_id`、`title`、`body`、`visibility`、`revealed_at`、`source_point_id`、`related_known_facts`。
- 响应示例：`{"data":{"items":[{"clue_id":"01J9Z3K4M7Q2R8T5V6W1XG0000","title":"船票残片","body":"票根上的时间被划去，只留下23:40。","visibility":"private","revealed_at":"2026-11-20T20:05:00Z","source_point_id":"01J9Z3K4M7Q2R8T5V6W1XV0000","related_known_facts":[]}],"next_cursor":null}}`
- 权限规则：只返回已揭示且 viewer 可见的线索；公共查询不得返回私人线索正文或未触发门控。
- 错误码：`GAME_NOT_FOUND`、`FORBIDDEN_VIEW`、`INVALID_CURSOR`。
- 幂等性：GET 安全可重试。

#### M06 提交质询

- 方法与 URL：`POST /api/v1/games/{game_id}/murder/questions`
- 鉴权/权限：`game_player`；阶段允许质询，目标在线或可被系统记录。
- 请求字段：`target_player_id`、`question_text`、`visibility=public|private`、`expected_version`。
- 请求示例：`{"target_player_id":"01J9Z3K4M7Q2R8T5V6W1X60000","question_text":"你23:40在哪里？","visibility":"public","expected_version":20}`
- 响应结构：`question`、`event_id`、`event_seq`、`state_version`、`answer_due_by`。
- 响应示例：`{"data":{"question":{"id":"01J9Z3K4M7Q2R8T5V6W1X70000","from_player_id":"01J9Z3K4M7Q2R8T5V6W1X80000","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","text":"你23:40在哪里？","visibility":"public"},"event_id":"01J9Z3K4M7Q2R8T5V6W1XA0000","event_seq":201,"state_version":21,"answer_due_by":"2026-11-21T10:06:00Z"}}`
- 权限规则：目标、阶段、频道范围由服务端校验；私密质询仅双方和授权主持人可见。
- 错误码：`TARGET_INVALID`、`PHASE_NOT_ALLOWED`、`QUESTION_LIMIT_EXCEEDED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一质询事件。

#### M07 提交或修改投票

- 方法与 URL：`POST /api/v1/games/{game_id}/murder/votes`
- 鉴权/权限：`game_player`；有投票权，且处于投票窗口。
- 请求字段：`ballot_type=murder_final`、`target_role_id`、`target_player_id`、`abstain`、`expected_version`。
- 请求示例：`{"ballot_type":"murder_final","target_role_id":"01J9Z3K4M7Q2R8T5V6W1XF0000","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X60000","abstain":false,"expected_version":30}`
- 响应结构：`vote`、`progress`、`vote_visibility`、`state_version`。
- 响应示例：`{"data":{"vote":{"round_key":"final_vote","voter_player_id":"01J9Z3K4M7Q2R8T5V6W1X40000","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X60000","abstain":false},"progress":{"submitted":5,"required":6},"vote_visibility":"count_only","state_version":31}}`
- 权限规则：每局每轮每人一个当前有效投票；改票由规则决定；投票结果揭示方式按模板。
- 错误码：`PHASE_NOT_ALLOWED`、`NO_VOTING_RIGHT`、`TARGET_INVALID`、`VOTE_CLOSED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键返回原投票；重复改票需新幂等键和新版本。

#### M08 查看票型

- 方法与 URL：`GET /api/v1/games/{game_id}/murder/votes`
- 鉴权/权限：`game_player`；仅按阶段和规则返回进度或明细。
- 请求字段：Query `round_key`、`view=progress|detail`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/murder/votes?round_key=final_vote&view=progress`
- 响应结构：`round_key`、`submitted_count`、`required_count`、`revealed`、`tally` 或 `votes`。
- 响应示例：`{"data":{"round_key":"final_vote","submitted_count":5,"required_count":6,"revealed":false,"tally":null,"votes":null}}`
- 权限规则：未结束且模板禁止公开票型时只返回进度；结算后用结果/复盘接口读取。
- 错误码：`GAME_NOT_FOUND`、`ROUND_NOT_FOUND`、`FORBIDDEN_VIEW`。
- 幂等性：GET 安全可重试。

### 6.2 狼人杀

#### W01 获取我的身份与技能视图

- 方法与 URL：`GET /api/v1/games/{game_id}/werewolf/me`
- 鉴权/权限：`game_player`；身份已分配。
- 请求字段：Query `include=identity,faction,skills,known_results,available_actions`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/werewolf/me?include=identity,skills,known_results,available_actions`
- 响应结构：`player_id`、`identity`、`alive`、`skills`、`known_results`、`available_actions`；狼人额外返回本阵营成员。
- 响应示例：`{"data":{"player_id":"01J9Z3K4M7Q2R8T5V6W1X70000","identity":{"role_name":"预言家","faction":"good","revealed":false},"alive":true,"skills":[{"skill_key":"seer_check","remaining_uses":null}],"known_results":[{"round_no":1,"target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","result":"wolf","visibility":"private"}],"available_actions":["seer_check","speak","vote"]}}`
- 权限规则：AI Player 与真人使用同一投影；不返回完整身份表；狼人阵营信息只给狼队。
- 错误码：`GAME_NOT_FOUND`、`ROLE_NOT_BOUND`、`GAME_NOT_STARTED`、`NOT_GAME_PLAYER`。
- 幂等性：GET 安全可重试。

#### W02 获取夜晚行动上下文

- 方法与 URL：`GET /api/v1/games/{game_id}/werewolf/night-context`
- 鉴权/权限：`game_player`；夜晚阶段且当前身份有行动窗口。
- 请求字段：Query `round_no`、`phase_key`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/werewolf/night-context?round_no=2&phase_key=night_actions`
- 响应结构：`can_act`、`action_window`、`allowed_skills`、`legal_targets`、`submitted`、`deadline_at`。
- 响应示例：`{"data":{"can_act":true,"action_window":"seer","allowed_skills":[{"skill_key":"seer_check","target_scope":"alive_players","self_allowed":false}],"legal_targets":[{"player_id":"01J9Z3K4M7Q2R8T5V6W1X80000","seat_no":5},{"player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","seat_no":7}],"submitted":false,"deadline_at":"2026-11-20T20:10:00Z"}}`
- 权限规则：目标列表按身份技能过滤；不能暴露其他角色身份、夜间结果或完整行动状态。
- 错误码：`PHASE_NOT_ALLOWED`、`NO_ACTIVE_SKILL`、`NIGHT_ACTION_WINDOW_CLOSED`、`FORBIDDEN_VIEW`。
- 幂等性：GET 安全可重试。

#### W03 提交夜间行动

- 方法与 URL：`POST /api/v1/games/{game_id}/werewolf/night-actions`
- 鉴权/权限：`game_player`；技能可用、目标合法且当前行动窗口打开。
- 请求字段：`skill_key`、`target_player_id`、`expected_version`、`client_request_id`。
- 请求示例：`{"skill_key":"seer_check","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","expected_version":22,"client_request_id":"wolf_act_01"}`
- 响应结构：`intent_id`、`status=accepted|resolved`、`submitted_action`、`state_version`、`event_range`。
- 响应示例：`{"data":{"intent_id":"intent_11...","status":"accepted","submitted_action":{"skill_key":"seer_check","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000"},"state_version":23,"event_range":{"start_seq":220,"end_seq":220}}}`
- 权限规则：夜间行动批量排序后结算；提交阶段不返回其他角色行动；AI 与真人同一 action 路径。
- 错误码：`PHASE_NOT_ALLOWED`、`SKILL_ALREADY_USED`、`TARGET_INVALID`、`NIGHT_ACTION_WINDOW_CLOSED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键返回原行动，不重复结算技能。

#### W04 提交白天发言

- 方法与 URL：`POST /api/v1/games/{game_id}/werewolf/speeches`
- 鉴权/权限：`game_player`；存活、轮到本人发言或允许补充发言。
- 请求字段：`round_no`、`speech_text`、`expected_version`。
- 请求示例：`{"round_no":2,"speech_text":"我认为7号昨晚的说法有矛盾。","expected_version":24}`
- 响应结构：`message`、`speech_order`、`next_player_id`、`state_version`。
- 响应示例：`{"data":{"message":{"id":"msg_31...","text":"我认为7号昨晚的说法有矛盾。","speaker_player_id":"01J9Z3K4M7Q2R8T5V6W1X70000"},"speech_order":{"current":4,"next":5},"next_player_id":"01J9Z3K4M7Q2R8T5V6W1X80000","state_version":25}}`
- 权限规则：仅本轮授权发言者；发言文本按公开发言投影；AI 发言不能包含越权身份信息。
- 错误码：`NOT_YOUR_TURN`、`PLAYER_ELIMINATED`、`SPEECH_TOO_LONG`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键返回原发言，不重复广播。

#### W05 提交或修改投票

- 方法与 URL：`POST /api/v1/games/{game_id}/werewolf/votes`
- 鉴权/权限：`game_player`；存活且有投票权，投票窗口打开。
- 请求字段：`ballot_type=exile|pk`、`target_player_id`、`abstain`、`round_no`、`expected_version`。
- 请求示例：`{"ballot_type":"exile","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","abstain":false,"round_no":2,"expected_version":32}`
- 响应结构：`vote`、`progress`、`state_version`。
- 响应示例：`{"data":{"vote":{"round_key":"day_2_exile","voter_player_id":"01J9Z3K4M7Q2R8T5V6W1X70000","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","abstain":false},"progress":{"submitted":6,"required":9},"state_version":33}}`
- 权限规则：每轮每人一个当前有效票；平票进入规则模板规定流程；不提前泄露票型。
- 错误码：`PHASE_NOT_ALLOWED`、`NO_VOTING_RIGHT`、`PLAYER_ELIMINATED`、`VOTE_CLOSED`、`TARGET_INVALID`。
- 幂等性：`Idempotency-Key` 必需；重复同票返回原结果。

#### W06 提交遗言

- 方法与 URL：`POST /api/v1/games/{game_id}/werewolf/last-words`
- 鉴权/权限：`game_player`；当前玩家刚被淘汰且模板允许遗言。
- 请求字段：`death_event_id`、`text`、`expected_version`。
- 请求示例：`{"death_event_id":"evt_401","text":"猎人请相信我，7号是狼。","expected_version":41}`
- 响应结构：`message`、`next_phase`、`state_version`。
- 响应示例：`{"data":{"message":{"id":"msg_44...","text":"猎人请相信我，7号是狼。"},"next_phase":"skill_chain","state_version":42}}`
- 权限规则：只有可遗言死亡玩家；不能冒用他人死亡事件；内容公开范围由模板决定。
- 错误码：`LAST_WORDS_NOT_ALLOWED`、`DEATH_EVENT_MISMATCH`、`LAST_WORDS_ALREADY_SUBMITTED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；一人一次，重复请求返回原遗言。

#### W07 触发身份技能

- 方法与 URL：`POST /api/v1/games/{game_id}/werewolf/skills`
- 鉴权/权限：`game_player`；技能触发器、存活/死亡条件与目标合法。
- 请求字段：`skill_key`、`target_player_id`、`trigger_event_id`、`confirmation`、`expected_version`。
- 请求示例：`{"skill_key":"hunter_shoot","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","trigger_event_id":"evt_401","confirmation":true,"expected_version":42}`
- 响应结构：`intent_id`、`status`、`skill_result`（按投影）、`skill_chain`、`state_version`、`event_range`。
- 响应示例：`{"data":{"intent_id":"intent_51...","status":"executed","skill_result":{"skill_key":"hunter_shoot","target_player_id":"01J9Z3K4M7Q2R8T5V6W1X90000","result":"pending_resolution"},"skill_chain":["hunter_shoot"],"state_version":43,"event_range":{"start_seq":402,"end_seq":403}}}`
- 权限规则：技能链由 Go 执行，防循环；AI 和真人同路径；未公开技能结果不返回其他玩家。
- 错误码：`SKILL_NOT_AVAILABLE`、`SKILL_TRIGGER_INVALID`、`TARGET_INVALID`、`SKILL_CHAIN_CONFLICT`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同技能同一事件链最多触发一次。

### 6.3 RPG

#### P01 获取当前场景上下文

- 方法与 URL：`GET /api/v1/games/{game_id}/rpg/context`
- 鉴权/权限：`game_player`；当前场景有效。
- 请求字段：Query `include=scene,map,party,tasks,inventory,available_actions`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/rpg/context?include=scene,map,party,tasks,available_actions`
- 响应结构：`scene`、`map`、`party`、`tasks`、`known_npcs`、`available_actions`、`state_version`。
- 响应示例：`{"data":{"scene":{"id":"01J9Z3K4M7Q2R8T5V6W1X8W378","title":"黑市下层","description":"潮湿的通道通向..."},"map":{"current":"01J9Z3K4M7Q2R8T5V6W1X8W378","exits":[{"to":"01J9Z3K4M7Q2R8T5V6W1XFRQHZ","label":"地下水道"}]},"party":[{"player_id":"01J9Z3K4M7Q2R8T5V6W1X50000","location":"01J9Z3K4M7Q2R8T5V6W1XFRQHZ"}],"tasks":[{"id":"01J9Z3K4M7Q2R8T5V6W1XEAP1C","title":"寻找失踪的弟弟","status":"active"}],"available_actions":["move","explore","talk","use_item"]}}`
- 权限规则：只返回当前队伍可见场景、地图边、任务和物品；NPC 私有知识和未发现出口不返回。
- 错误码：`GAME_NOT_FOUND`、`NOT_GAME_PLAYER`、`SCENE_STATE_UNAVAILABLE`、`FORBIDDEN_VIEW`。
- 幂等性：GET 安全可重试。

#### P02 移动

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/moves`
- 鉴权/权限：`game_player`；目标场景可达且队伍/道具条件满足。
- 请求字段：`target_scene_id`、`expected_version`、`client_request_id`。
- 请求示例：`{"target_scene_id":"01J9Z3K4M7Q2R8T5V6W1XFRQHZ","expected_version":50,"client_request_id":"move_01"}`
- 响应结构：`intent_id`、`status`、`scene`、`party_location`、`state_version`、`event_range`。
- 响应示例：`{"data":{"intent_id":"intent_61...","status":"executed","scene":{"id":"01J9Z3K4M7Q2R8T5V6W1XFRQHZ","title":"地下水道"},"party_location":{"player_id":"01J9Z3K4M7Q2R8T5V6W1X40000","scene_id":"01J9Z3K4M7Q2R8T5V6W1XFRQHZ"},"state_version":51,"event_range":{"start_seq":501,"end_seq":502}}}`
- 权限规则：不可达、资源不足或场景冲突返回拒绝；多人冲突只一个事务成功。
- 错误码：`TARGET_UNREACHABLE`、`RESOURCE_NOT_ENOUGH`、`SCENE_VERSION_CONFLICT`、`PHASE_NOT_ALLOWED`。
- 幂等性：`Idempotency-Key` 必需；同键不重复移动。

#### P03 探索

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/explores`
- 鉴权/权限：`game_player`；当前场景允许探索。
- 请求字段：`target_scene_id` 可选、`search_scope`、`expected_version`。
- 请求示例：`{"search_scope":"hidden_exit","expected_version":51}`
- 响应结构：`intent_id`、`discoveries`、`clue_ids`、`scene_updates`、`state_version`、`event_range`。
- 响应示例：`{"data":{"intent_id":"intent_62...","discoveries":[{"kind":"hidden_exit","scene_id":"01J9Z3K4M7Q2R8T5V6W1XVZ1ZD","visibility":"party"}],"clue_ids":["clue_11..."],"scene_updates":[],"state_version":52,"event_range":{"start_seq":503,"end_seq":504}}}`
- 权限规则：探索结果只能来自已配置事件或受控动态事件；关键新出口/道具必须经世界引擎。
- 错误码：`PHASE_NOT_ALLOWED`、`SCENE_STATE_UNAVAILABLE`、`EXPLORE_QUOTA_EXHAUSTED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键不重复产出探索奖励。

#### P04 解析自由文本行动

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/actions/parse`
- 鉴权/权限：`game_player`；游戏允许自由文本。
- 请求字段：`text`、`scene_id`、`base_version`、`locale`。
- 请求示例：`{"text":"我检查木箱后面有没有暗道","scene_id":"01J9Z3K4M7Q2R8T5V6W1X8W378","base_version":52,"locale":"zh-CN"}`
- 响应结构：`ai_run_id`、`candidates`、`requires_clarification`、`no_state_change`。
- 响应示例：`{"data":{"ai_run_id":"01J9Z3K4M7Q2R8T5V6W1X2RAP6","candidates":[{"action_type":"explore","target":"wooden_crate","confidence":0.89},{"action_type":"observe","target":"wooden_crate","confidence":0.63}],"requires_clarification":false,"no_state_change":true}}`
- 权限规则：解析不改变状态；候选不得包含非法目标或世界规则外动作；低置信度要求澄清。
- 错误码：`PHASE_NOT_ALLOWED`、`AI_TIMEOUT`、`AI_SAFETY_BLOCKED`、`INVALID_SCENE`、`RATE_LIMITED`。
- 幂等性：不改变权威状态，但建议携带 `Idempotency-Key` 复用 AI 结果；同键同文本返回同一候选。

#### P05 执行结构化 RPG 动作

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/actions`
- 鉴权/权限：`game_player`；动作在当前场景、阶段和资源下合法。
- 请求字段：`action_type=move|observe|interact|use_item|assist|attack|trade|wait`、`target`、`parameters`、`expected_version`。
- 请求示例：`{"action_type":"interact","target":{"type":"scene_entity","id":"crate_01"},"parameters":{"intent":"inspect_back"},"expected_version":52}`
- 响应结构：`intent_id`、`status`、`resolved_events`、`state_changes`（按 viewer）、`state_version`、`event_range`。
- 响应示例：`{"data":{"intent_id":"intent_63...","status":"executed","resolved_events":[{"event_id":"evt_505","event_type":"rpg.entity_inspected","summary":"木箱后没有暗道，但发现一枚旧钥匙。"}],"state_changes":[{"kind":"item_added","item":{"id":"01J9Z3K4M7Q2R8T5V6W1X0JZ63","name":"旧钥匙"},"visibility":"player"}],"state_version":53,"event_range":{"start_seq":505,"end_seq":505}}}`
- 权限规则：AI 只能返回候选，最终合法性由 RPG Engine 判定；关键世界状态变更必须有公共/作用域事件。
- 错误码：`ACTION_NOT_ALLOWED`、`TARGET_INVALID`、`RESOURCE_NOT_ENOUGH`、`STATE_VERSION_CONFLICT`、`ACTION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键返回原决策和事件范围。

#### P06 NPC 对话

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/npcs/{npc_id}/dialogues`
- 鉴权/权限：`game_player`；NPC 在场、场景有效、对话范围允许。
- 请求字段：`text`、`scene_id`、`expected_version`、`conversation_visibility=private|scene`。
- 请求示例：`{"text":"你见过一个戴银戒指的人吗？","scene_id":"01J9Z3K4M7Q2R8T5V6W1X8W378","expected_version":53,"conversation_visibility":"private"}`
- 响应结构：`ai_run_id`、`dialogue`、`relationship_delta`、`memory_candidate_ref` 不直接暴露、`suggested_actions`、`safety_status`。
- 响应示例：`{"data":{"ai_run_id":"01J9Z3K4M7Q2R8T5V6W1X1NKJE","dialogue":{"message_id":"01J9Z3K4M7Q2R8T5V6W1X7B1M8","npc_id":"01J9Z3K4M7Q2R8T5V6W1XFSZJA","text":"银戒指？上周有个商人在码头问过同样的东西。","speaker":"黑市商人"},"relationship_delta":{"trust":1},"suggested_actions":["ask_about_merchant"],"safety_status":"passed"}}`
- 权限规则：只注入该 NPC 的授权记忆、关系和场景切片；世界动作必须另走 P05；不返回 NPC 私有知识或记忆原文。
- 错误码：`NPC_NOT_FOUND`、`NPC_NOT_PRESENT`、`AI_TIMEOUT`、`AI_SAFETY_BLOCKED`、`DIALOGUE_WINDOW_CLOSED`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键不重复扣调用预算或重复写对话事件。

#### P07 NPC 对话历史

- 方法与 URL：`GET /api/v1/games/{game_id}/rpg/npcs/{npc_id}/dialogues`
- 鉴权/权限：`game_player`；只读取参与或旁听可见对话。
- 请求字段：Query `conversation_visibility`、`after_message_id`、`cursor`、`limit`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/rpg/npcs/01J9Z3K4M7Q2R8T5V6W1XFSZJA/dialogues?limit=30`
- 响应结构：分页 `dialogue_messages`、`visibility`、`relationship_summary`、`next_cursor`。
- 响应示例：`{"data":{"items":[{"message_id":"01J9Z3K4M7Q2R8T5V6W1X7B1M8","speaker":"npc","text":"银戒指？上周有个商人在码头问过同样的东西。","visibility":"private"}],"visibility":"private","relationship_summary":{"trust":12,"fear":0},"next_cursor":null}}`
- 权限规则：私聊只给参与双方；公开场景对话按场景可见；不得返回 NPC 全部记忆。
- 错误码：`NPC_NOT_FOUND`、`FORBIDDEN_VIEW`、`INVALID_CURSOR`。
- 幂等性：GET 安全可重试。

#### P08 查询任务

- 方法与 URL：`GET /api/v1/games/{game_id}/rpg/tasks`
- 鉴权/权限：`game_player`。
- 请求字段：Query `status=available|active|completed|failed|expired`、`cursor`、`limit`。
- 请求示例：`GET /api/v1/games/01J9Z3K4M7Q2R8T5V6W1X10000/rpg/tasks?status=active`
- 响应结构：分页 `task`：`task_id`、`title`、`status`、`public_description`、`progress`、`visibility`、`expires_at`。
- 响应示例：`{"data":{"items":[{"task_id":"01J9Z3K4M7Q2R8T5V6W1XEAP1C","title":"寻找失踪的弟弟","status":"active","public_description":"寻找在雾港失踪的船员","progress":{"current":1,"target":3},"visibility":"player","expires_at":null}],"next_cursor":null}}`
- 权限规则：只返回本人/队伍可接任务；隐藏结局和 NPC 私有目标不返回。
- 错误码：`GAME_NOT_FOUND`、`FORBIDDEN_VIEW`、`VALIDATION_ERROR`。
- 幂等性：GET 安全可重试。

#### P09 接受或放弃任务

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/tasks/{task_id}/actions`
- 鉴权/权限：`game_player`；任务可用/进行中且满足前置条件。
- 请求字段：`action=accept|abandon`、`expected_version`。
- 请求示例：`{"action":"accept","expected_version":53}`
- 响应结构：`task`、`accepted`、`state_version`、`event_range`。
- 响应示例：`{"data":{"task":{"task_id":"01J9Z3K4M7Q2R8T5V6W1XEAP1C","status":"active"},"accepted":true,"state_version":54,"event_range":{"start_seq":506,"end_seq":506}}}`
- 权限规则：服务端校验任务可见性、前置事件和队伍状态；不能通过 task_id 枚举他人任务。
- 错误码：`TASK_NOT_FOUND`、`TASK_NOT_AVAILABLE`、`TASK_PREREQUISITE_MISSING`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；重复 accept 返回当前 active 状态。

#### P10 使用物品

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/items/use`
- 鉴权/权限：`game_player`；物品在背包、场景和规则允许使用。
- 请求字段：`item_id`、`target_type`、`target_id`、`quantity`、`expected_version`。
- 请求示例：`{"item_id":"01J9Z3K4M7Q2R8T5V6W1X0JZ63","target_type":"scene_entity","target_id":"door_01","quantity":1,"expected_version":54}`
- 响应结构：`intent_id`、`status`、`effects`、`inventory_delta`、`state_version`、`event_range`。
- 响应示例：`{"data":{"intent_id":"intent_71...","status":"executed","effects":[{"kind":"door_unlocked","target_id":"door_01"}],"inventory_delta":{"item_id":"01J9Z3K4M7Q2R8T5V6W1X0JZ63","quantity_delta":-1},"state_version":55,"event_range":{"start_seq":507,"end_seq":507}}}`
- 权限规则：只能使用本人/队伍授权物品；交易、给物和关键世界变更不得仅由 AI 文本决定。
- 错误码：`ITEM_NOT_FOUND`、`ITEM_NOT_OWNED`、`TARGET_INVALID`、`ITEM_EFFECT_UNAVAILABLE`、`STATE_VERSION_CONFLICT`。
- 幂等性：`Idempotency-Key` 必需；同键不重复扣物品。

#### P11 提交剧情选择

- 方法与 URL：`POST /api/v1/games/{game_id}/rpg/story-choices`
- 鉴权/权限：`game_player`；当前剧情节点/场景存在且选择可见。
- 请求字段：`scene_id`、`choice_id`、`expected_version`、`confirmation`。
- 请求示例：`{"scene_id":"01J9Z3K4M7Q2R8T5V6W1XMBN6R","choice_id":"choice_destroy_gate","expected_version":60,"confirmation":true}`
- 响应结构：`choice_result`、`public_world_changes`、`personal_changes`、`state_version`、`event_range`。
- 响应示例：`{"data":{"choice_result":{"choice_id":"choice_destroy_gate","summary":"你关闭了水闸，城市警报升高。"},"public_world_changes":[{"variable":"city_alert","from":3,"to":5}],"personal_changes":[{"kind":"reputation","faction":"harbor_guard","delta":-2}],"state_version":61,"event_range":{"start_seq":511,"end_seq":514}}}`
- 权限规则：结局/世界变化条件由世界引擎裁决；AI 只能叙述已成立结果。
- 错误码：`CHOICE_NOT_AVAILABLE`、`SCENE_VERSION_CONFLICT`、`PARTY_CONFIRMATION_REQUIRED`、`ENDING_NOT_AVAILABLE`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一 choice 结果。

---

## 7. AI 与社区接口契约

### 7.1 AI

#### A01 查询脱敏 AI Run

- 方法与 URL：`GET /api/v1/ai-runs/{ai_run_id}`
- 鉴权/权限：生成/复盘 Run 的 owner；游戏内 Run 的当前 game player；事故审计员。
- 请求字段：Path `ai_run_id`；Query `view=user|author|auditor`。
- 请求示例：`GET /api/v1/ai-runs/01J9Z3K4M7Q2R8T5V6W1XT0000?view=user`
- 响应结构：`ai_run_id`、`task_type`、`status`、`model_summary`、`prompt_version_summary`、`timing`、`usage`、`safety`、`accepted_status`。
- 响应示例：`{"data":{"ai_run_id":"01J9Z3K4M7Q2R8T5V6W1XT0000","task_type":"npc_turn","status":"succeeded","model_summary":{"provider":"provider_a","model_key":"small-fast"},"prompt_version_summary":{"prompt_key":"npc.dialogue","version_no":7},"timing":{"duration_ms":832},"usage":{"input_tokens":1120,"output_tokens":260,"cost_amount":0.0042,"currency":"USD"},"safety":{"status":"passed","flags":[]},"accepted_status":"accepted"}}`
- 权限规则：普通用户不获得完整 Prompt、context manifest 正文、工具参数、模型原始输出和内部思维链；审计视图仍需最小权限与审计。
- 错误码：`AI_RUN_NOT_FOUND`、`FORBIDDEN_VIEW`、`AUDIT_APPROVAL_REQUIRED`。
- 幂等性：GET 安全可重试。

### 7.2 社区

#### C01 评分与短评列表

- 方法与 URL：`GET /api/v1/scripts/{script_id}/reviews`
- 鉴权/权限：`anonymous/session`；公开版本。
- 请求字段：`cursor`、`limit`、`sort=helpful|newest`、`hide_spoilers=true`。
- 请求示例：`GET /api/v1/scripts/01J9Z3K4M7Q2R8T5V6W1X30000/reviews?sort=newest&hide_spoilers=true`
- 响应结构：分页 `review`：`review_id`、`rating`、`text`、`contains_spoiler`、`author_public_name`、`created_at`、`moderation_status`。
- 响应示例：`{"data":{"items":[{"review_id":"01J9Z3K4M7Q2R8T5V6W1XAKDZT","rating":5,"text":"线索链完整。","contains_spoiler":false,"author_public_name":"Bob","created_at":"2026-11-20T12:00:00Z","moderation_status":"published"}],"next_cursor":null}}`
- 权限规则：只返回公开/已审核内容；剧透评论默认折叠；作者隐私按公开主页规则。
- 错误码：`CONTENT_NOT_FOUND`、`FORBIDDEN_VIEW`、`INVALID_CURSOR`。
- 幂等性：GET 安全可重试。

#### C02 提交评分/短评

- 方法与 URL：`POST /api/v1/scripts/{script_id}/reviews`
- 鉴权/权限：`session_csrf`；用户完成过该版本至少一局。
- 请求字段：`script_version_id`、`game_session_id`、`rating`、`review_text`、`contains_spoiler`。
- 请求示例：`{"script_version_id":"01J9Z3K4M7Q2R8T5V6W1XW0000","game_session_id":"01J9Z3K4M7Q2R8T5V6W1X10000","rating":5,"review_text":"线索链完整。","contains_spoiler":false}`
- 响应结构：`review_id`、`status=pending|published`、`rating_summary`。
- 响应示例：`{"data":{"review_id":"01J9Z3K4M7Q2R8T5V6W1XAKDZT","status":"pending","rating_summary":{"score":4.6,"count":29}}}`
- 权限规则：每用户每版本一条评分；必须能验证完成对局；剧透标记不可绕过。
- 错误码：`REVIEW_NOT_ALLOWED`、`GAME_NOT_COMPLETED`、`REVIEW_ALREADY_EXISTS`、`CONTENT_LENGTH_INVALID`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一 review；更新评分使用唯一资源语义。

#### C03 评论列表

- 方法与 URL：`GET /api/v1/scripts/{script_id}/comments`
- 鉴权/权限：`anonymous/session`。
- 请求字段：`cursor`、`limit`、`sort=oldest|newest`、`parent_comment_id`。
- 请求示例：`GET /api/v1/scripts/01J9Z3K4M7Q2R8T5V6W1X30000/comments?sort=newest&limit=30`
- 响应结构：分页 `comment`：`comment_id`、`parent_comment_id`、`body`、`contains_spoiler`、`author_public_name`、`created_at`、`status`。
- 响应示例：`{"data":{"items":[{"comment_id":"01J9Z3K4M7Q2R8T5V6W1XR7VPF","parent_comment_id":null,"body":"这个主题很适合六人局。","contains_spoiler":false,"author_public_name":"Carol","created_at":"2026-11-20T13:00:00Z","status":"published"}],"next_cursor":null}}`
- 权限规则：只返回可见评论；剧透默认折叠；被删除评论显示不可用状态，不返回原正文。
- 错误码：`CONTENT_NOT_FOUND`、`INVALID_CURSOR`。
- 幂等性：GET 安全可重试。

#### C04 发表评论

- 方法与 URL：`POST /api/v1/scripts/{script_id}/comments`
- 鉴权/权限：`session_csrf`；公开内容可评论，未开放社区功能返回 `FEATURE_DISABLED`。
- 请求字段：`parent_comment_id`、`body`、`contains_spoiler`。
- 请求示例：`{"parent_comment_id":null,"body":"这个主题很适合六人局。","contains_spoiler":false}`
- 响应结构：`comment_id`、`status`、`created_at`、`moderation_hint`。
- 响应示例：`{"data":{"comment_id":"01J9Z3K4M7Q2R8T5V6W1XR7VPF","status":"pending","created_at":"2026-11-20T13:00:00Z","moderation_hint":"under_review"}}`
- 权限规则：服务端做内容审核、频率限制、父评论归属校验；不能评论无权访问内容。
- 错误码：`CONTENT_NOT_FOUND`、`FEATURE_DISABLED`、`COMMENT_TOO_LONG`、`RATE_LIMITED`、`MODERATION_REJECTED`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一评论。

#### C05 设置或取消收藏

- 方法与 URL：`PUT /api/v1/scripts/{script_id}/favorite`
- 鉴权/权限：`session_csrf`；内容公开或本人可访问。
- 请求字段：`favorited`、`collection_name`。
- 请求示例：`{"favorited":true,"collection_name":"周末局"}`
- 响应结构：`script_id`、`favorited`、`collection_name`、`updated_at`。
- 响应示例：`{"data":{"script_id":"01J9Z3K4M7Q2R8T5V6W1X30000","favorited":true,"collection_name":"周末局","updated_at":"2026-11-20T14:00:00Z"}}`
- 权限规则：只能操作本人收藏；收藏不扩大内容访问权。
- 错误码：`CONTENT_NOT_FOUND`、`FORBIDDEN_VIEW`、`VALIDATION_ERROR`。
- 幂等性：`Idempotency-Key` 必需；相同 favorited 值返回当前状态。

#### C06 创建分享链接

- 方法与 URL：`POST /api/v1/share-links`
- 鉴权/权限：`session_csrf`；对象可分享且不扩大原权限。
- 请求字段：`object_type=script|room|replay`、`object_id`、`max_uses`、`expires_in_seconds`。
- 请求示例：`{"object_type":"script","object_id":"01J9Z3K4M7Q2R8T5V6W1X30000","max_uses":100,"expires_in_seconds":604800}`
- 响应结构：`share_link_id`、`url`、`expires_at`、`max_uses`、`visibility_warning`。
- 响应示例：`{"data":{"share_link_id":"01J9Z3K4M7Q2R8T5V6W1X2VQQA","url":"https://mysteryverse.example/s/01J9Z3K4M7Q2R8T5V6W1X2VQQA","expires_at":"2026-11-27T14:00:00Z","max_uses":100,"visibility_warning":"does_not_expand_original_permissions"}}`
- 权限规则：私人房间/复盘链接仍要求登录、成员和角色校验；不能通过分享绕过 D2-D4。
- 错误码：`OBJECT_NOT_SHAREABLE`、`FORBIDDEN_VIEW`、`RATE_LIMITED`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一链接。

#### C07 举报

- 方法与 URL：`POST /api/v1/reports`
- 鉴权/权限：`session_csrf`；举报对象必须可定位。
- 请求字段：`object_type`、`object_id`、`reason_code`、`details`、`evidence`。
- 请求示例：`{"object_type":"message","object_id":"01J9Z3K4M7Q2R8T5V6W1XP0000","reason_code":"harassment","details":"持续人身攻击","evidence":{"game_id":"01J9Z3K4M7Q2R8T5V6W1X10000","event_id":"evt_186"}}`
- 响应结构：`report_id`、`status=open`、`created_at`、`case_visibility`。
- 响应示例：`{"data":{"report_id":"01J9Z3K4M7Q2R8T5V6W1X2P0S3","status":"open","created_at":"2026-11-20T15:00:00Z","case_visibility":"reporter_can_view_status"}}`
- 权限规则：举报证据按最小权限存储；举报人不自动获得被举报私密内容或后台工单。
- 错误码：`OBJECT_NOT_FOUND`、`REASON_INVALID`、`DUPLICATE_REPORT`、`RATE_LIMITED`。
- 幂等性：`Idempotency-Key` 必需；同键返回同一 report。

---

## 8. WebSocket 协议

### 8.1 连接建立

1. 客户端通过 Session 调用 `POST /api/v1/realtime/tickets`。
2. 服务端校验账号、房间成员、有效席位和目标游戏。
3. 返回一次性 `ticket`，有效期建议 30 秒，绑定 `user_id + room_id + purpose + nonce`。
4. 浏览器建立连接：`wss://api.mysteryverse.example/ws/v1`。
5. 浏览器使用子协议携带版本和 ticket：

```javascript
const socket = new WebSocket(
  "wss://api.mysteryverse.example/ws/v1",
  ["mvws.v1", `mvws.ticket.${ticket}`]
);
```

6. 服务端验证 Origin、子协议、ticket、Session、房间成员和账号状态。
7. 服务端只回选 `mvws.v1` 子协议，并发送 `server.ready`。
8. 客户端发送 `room.subscribe`；已开局则再发送 `game.subscribe`。
9. 服务端返回 room/game snapshot 或已投影增量。
10. ticket 单次使用；握手失败、过期或用户退出后不可复用。

不要在 URL 中携带 ticket。若网关不支持子协议传 ticket，可改用首帧 `client.auth`，但服务端在认证成功前不得推送任何房间数据。

### 8.2 统一消息信封

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
    "clue": {
      "id": "01J9Z3K4M7Q2R8T5V6W1X90000",
      "title": "船票残片"
    },
    "scope": "player"
  }
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `event_id` | string | 全局唯一纯 26 字符 ULID；客户端首选去重键 |
| `event_type` | string | 稳定事件类型 |
| `stream` | string | `system`、`room`、`room_chat`、`game`、`ai_progress` 或 `ai_stream` |
| `stream_id` | string | 当前流实例的纯 ULID |
| `seq` | integer | 同一 `stream + stream_id` 内严格递增序号 |
| `schema_version` | integer | 正整数载荷契约版本 |
| `game_id` | string/null | 游戏级事件必填；等待房事件不得伪造 |
| `timestamp` | string | UTC RFC 3339 服务端事件时间 |
| `payload` | object | 已按 viewer 完成权限投影的 DTO |

| stream | `stream_id` | `seq` 来源 | `game_id` |
| --- | --- | --- | --- |
| `system` | `connection_id` | 连接控制序号 | 无 |
| `room` | `room_id` | `room_events.seq` | 无 |
| `room_chat` | `room_id` | `room_messages.chat_seq` | 无 |
| `game` | `game_id` | `game_events.seq` | 必填 |
| `ai_progress` | `generation_job_id` | 任务进度序号 | 内容生成任务无 |
| `ai_stream` | `ai_run_id` | 单次 Run 临时流序号 | 游戏内必填 |

约束：

- 同一 `stream + stream_id` 的 `seq` 单调递增；不同 stream 或实例不能直接比较序号。
- `room` 使用 `room_events.seq`；`room_chat` 使用 `room_messages.chat_seq`；`game` 使用 `game_events.seq`。
- `ai_progress` 和 `ai_stream` 不推进游戏状态版本，也不写入 `game_events.seq`。
- 禁止再使用 `session_id`、`sequence`、`messageId`、`streamId` 或 `projectionVersion` 作为同义协议字段。
- `payload` 不得包含 `secret_payload_id`、密文、完整身份表、完整 Prompt 或其他 viewer 无权字段。
- 客户端不能提交 `event_id`、`seq`、`schema_version`、`visibility` 或 `state_version` 来伪造服务端事件。

### 8.3 客户端到服务端消息

#### 8.3.1 `room.subscribe`

```json
{
  "event_type": "room.subscribe",
  "request_id": "01J9Z3K4M7Q2R8T5V6W1X20000",
  "schema_version": 1,
  "payload": {
    "room_id": "01J9Z3K4M7Q2R8T5V6W1X80000",
    "last_room_seq": 12
  }
}
```

服务端再次校验成员，返回 `room.snapshot` 或增量 `room.event`。

#### 8.3.2 `game.subscribe`

```json
{
  "event_type": "game.subscribe",
  "request_id": "01J9Z3K4M7Q2R8T5V6W1X30000",
  "schema_version": 1,
  "payload": {
    "game_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
    "last_game_seq": 184,
    "last_state_version": 17,
    "protocol_version": "mvws.v1"
  }
}
```

服务端校验当前席位、角色和 game 归属后返回 `game.snapshot` 或增量。

#### 8.3.3 `resync.request`

```json
{
  "event_type": "resync.request",
  "request_id": "01J9Z3K4M7Q2R8T5V6W1X40000",
  "schema_version": 1,
  "payload": {
    "game_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
    "last_room_seq": 12,
    "last_room_chat_seq": 18,
    "last_game_seq": 184,
    "last_state_version": 17,
    "reason": "seq_gap"
  }
}
```

大快照优先通过 HTTP `GET /api/v1/games/{game_id}/projection` 或 `POST /reconnect` 获取，不通过单条 WebSocket 消息塞入。

#### 8.3.4 `heartbeat.ping`

```json
{
  "event_type": "heartbeat.ping",
  "request_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "schema_version": 1,
  "payload": {
    "client_time": "2026-11-21T10:05:00Z"
  }
}
```

服务端返回 `heartbeat.pong`，包含 `server_time` 和当前 room/game seq。

#### 8.3.5 `message.ack`

```json
{
  "event_type": "message.ack",
  "request_id": "01J9Z3K4M7Q2R8T5V6W1X60000",
  "schema_version": 1,
  "payload": {
    "event_id": "01J9Z3K4M7Q2R8T5V6W1X70000"
  }
}
```

仅当事件类型契约声明需要 ACK 时发送；ACK 不代表事件可修改或可删除。

### 8.4 服务端到客户端消息

#### 8.4.1 `server.ready`

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X60000",
  "event_type": "server.ready",
  "stream": "system",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X70000",
  "seq": 1,
  "schema_version": 1,
  "timestamp": "2026-11-21T10:05:00Z",
  "payload": {
    "connection_id": "01J9Z3K4M7Q2R8T5V6W1X70000",
    "user_id": "01J9Z3K4M7Q2R8T5V6W1X80000",
    "room_id": "01J9Z3K4M7Q2R8T5V6W1X90000",
    "room_seq": 12,
    "room_chat_seq": 18,
    "game_seq": 184,
    "state_version": 17,
    "supported_protocol": "mvws.v1",
    "heartbeat_interval_ms": 25000
  }
}
```

#### 8.4.2 `room.snapshot`

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1XA0000",
  "event_type": "room.snapshot",
  "stream": "room",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X90000",
  "seq": 12,
  "schema_version": 1,
  "timestamp": "2026-11-21T10:05:01Z",
  "payload": {
    "room_id": "01J9Z3K4M7Q2R8T5V6W1X90000",
    "room": {
      "status": "waiting",
      "capacity": {"min": 6, "max": 6}
    },
    "members": [
      {
        "member_id": "01J9Z3K4M7Q2R8T5V6W1XB0000",
        "seat_no": 1,
        "nickname": "Alice",
        "ready": true,
        "room_role": "host"
      }
    ],
    "viewer_capabilities": ["ready"]
  }
}
```

#### 8.4.3 `game.snapshot`

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1XC0000",
  "event_type": "game.snapshot",
  "stream": "game",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "seq": 184,
  "schema_version": 1,
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "timestamp": "2026-11-21T10:05:02Z",
  "payload": {
    "state_version": 17,
    "game_type": "script_murder",
    "phase": {
      "key": "search_round_1",
      "deadline_at": "2026-11-21T10:08:00Z"
    },
    "public_state": {
      "progress": {"searched": 2, "required": 6}
    },
    "private_state": {
      "role_id": "01J9Z3K4M7Q2R8T5V6W1XD0000",
      "search_quota": 1
    },
    "available_actions": ["search", "chat", "vote"]
  }
}
```

### 8.5 事件类型清单

| `event_type` | stream | 默认 visibility | payload 摘要 |
| --- | --- | --- | --- |
| `room.player_joined` | room | room | 成员、席位、公开昵称 |
| `room.player_left` | room | room | 成员、原因类别、是否房主 |
| `room.player_ready` | room | room | 成员、ready 状态 |
| `room.message_received` | room_chat | room | 等待房聊天消息、发送者和审核状态 |
| `room.host_transferred` | room | room | 旧/新房主公开 ID |
| `room.closed` | room | room | 关闭原因 |
| `game.started` | game | room | game_id、模式、引擎版本、初始公开阶段 |
| `game.phase_changed` | game | room/role | 新阶段、计时、允许动作 |
| `game.state_updated` | game | player/role | viewer 可用的状态补丁和 `state_version` |
| `game.clue_unlocked` | game | public/role/player | 已投影线索，不含未授权正文 |
| `game.message_received` | game | room/role/player | 消息摘要、发送者公开信息、scope |
| `game.vote_submitted` | game | room/player | 投票进度；明细按模板 |
| `game.player_eliminated` | game | room/role | 规则允许的死亡/淘汰信息 |
| `game.vote_resolved` | game | room/role | 票型、平票和放逐结果 |
| `game.paused` | game | room/role | 暂停状态、剩余时间和状态版本 |
| `game.resumed` | game | room/role | 恢复状态、新 deadline 和状态版本 |
| `game.timer_updated` | game | room/role | 阶段 deadline、暂停剩余时间 |
| `game.skill_triggered` | game | role/role_group | 已投影技能结果 |
| `game.ended` | game | room | 结果可用、只读状态 |
| `game.result_ready` | game | room | result_id 和结果摘要入口 |
| `game.connection_recovered` | game | player | 重连完成、恢复到的版本/序号 |
| `game.incident_frozen` | game | room/audit | 冻结原因类别、支持入口 |
| `npc.dialogue_generated` | game | player/scene | 已安全检查的 NPC 台词与来源引用 |
| `ai.run.degraded` | ai_stream | player | 游戏内 AI 降级等级与可见提示码 |
| `ai.progress.degraded` | ai_progress | owner | 生成任务降级状态 |
| `ai.stream.chunk` | ai_stream | player | 展示用文本片段，非权威状态 |
| `ai.progress` | ai_progress | owner | 生成任务节点与进度 |
| `server.error` | system | player | 统一错误对象 |
| `server.resync_required` | system | player | 指定 room/room_chat/game 流需要重新获取快照 |

### 8.6 公共与私密事件

#### 房间公开广播示例

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "event_type": "room.player_joined",
  "stream": "room",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X20000",
  "seq": 13,
  "schema_version": 1,
  "timestamp": "2026-11-21T10:05:30Z",
  "payload": {
    "member": {
      "member_id": "01J9Z3K4M7Q2R8T5V6W1X30000",
      "seat_no": 4,
      "nickname": "Dave",
      "room_role": "player"
    }
  }
}
```

#### 公共线索示例

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X40000",
  "event_type": "game.clue_unlocked",
  "stream": "game",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "seq": 190,
  "schema_version": 1,
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "timestamp": "2026-11-21T10:06:00Z",
  "payload": {
    "clue": {
      "id": "01J9Z3K4M7Q2R8T5V6W1X60000",
      "title": "甲板血迹",
      "text": "血迹从船舱延伸到栏杆。",
      "scope": "public"
    }
  }
}
```

#### 私人线索示例

以下事件只推送给发现者，不能发到公共频道：

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X70000",
  "event_type": "game.clue_unlocked",
  "stream": "game",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "seq": 191,
  "schema_version": 1,
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "timestamp": "2026-11-21T10:06:05Z",
  "payload": {
    "clue": {
      "id": "01J9Z3K4M7Q2R8T5V6W1X80000",
      "title": "船票残片",
      "text": "票根上的时间被划去，只留下23:40。",
      "scope": "private"
    }
  }
}
```

私密广播规则：

- 服务端先执行对象 ACL、字段 ACL、reveal gate 和 viewer 投影。
- 私密事件通过用户定向频道或角色组定向 pub/sub 发送。
- 浏览器不直接订阅 Redis Channel，也不靠客户端频道名隔离。
- 房主、创作者和普通运营不会因为连接同一房间获得私密事件。
- 狼人频道只向规则允许的狼人角色/阵营投影。
- 私聊 `game.message_received` 只发给参与双方和明确授权旁听者。

#### 私聊消息示例

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X90000",
  "event_type": "game.message_received",
  "stream": "game",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "seq": 192,
  "schema_version": 1,
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X50000",
  "timestamp": "2026-11-21T10:06:10Z",
  "payload": {
    "message": {
      "message_id": "01J9Z3K4M7Q2R8T5V6W1XA0000",
      "sender": {
        "type": "user",
        "player_id": "01J9Z3K4M7Q2R8T5V6W1XB0000",
        "display_name": "Alice"
      },
      "message_type": "private",
      "text": "你昨晚在哪里？",
      "scope": "player"
    }
  }
}
```

该事件只发送给 `01J9Z3K4M7Q2R8T5V6W1X40000`、目标参与者 `01J9Z3K4M7Q2R8T5V6W1X50000` 和规则明确授权的旁听主体，不发送到房间公共频道。

### 8.7 心跳与连接健康

- WebSocket 协议层 Ping/Pong 与应用层 `heartbeat.ping/pong` 同时启用。
- 服务端建议 25 秒发送 Ping；客户端 20-30 秒无有效响应进入重连。
- 连接空闲超过 60 秒可关闭，但玩家席位保留配置的离线宽限。
- 心跳只更新 presence，不改变游戏状态。
- 同一账号是否允许多连接由产品策略决定；同一房间仅一个玩家席位。
- Session 撤销后服务端关闭连接或后续写操作返回 `SESSION_REVOKED`。

### 8.8 重连与消息补偿

客户端保存：

- `last_room_seq`
- `last_room_chat_seq`
- `last_game_seq`
- `last_state_version`
- `last_event_id`
- 协议版本和当前页面

重连流程：

1. 重新确认 Session；必要时重新登录。
2. 调用 `POST /api/v1/realtime/tickets` 获取新 ticket。
3. 新建 WebSocket，发送 `room.subscribe` / `room_chat.subscribe` / `game.subscribe` 和对应最后序号。
4. 服务端重新校验成员、席位、角色和 game 归属。
5. 差额小且可见序列连续：服务端返回 `room.snapshot`/`game.snapshot` + 增量事件。
6. 差额大、事件已归档或权限视图改变：只返回新 snapshot。
7. 客户端清空旧游戏投影后应用 snapshot。
8. 未确认操作使用原 `Idempotency-Key` 通过 REST 重试，服务端返回原结果。
9. 收到较小 `seq` 忽略；收到未来序号缺口立即发送 `resync.request`。

消息补偿顺序：

```text
鉴权
→ 房间/席位
→ 选最近可信快照
→ 读取 last_seq + 1 之后可见事件
→ 投影
→ 返回 snapshot + delta 或 only snapshot
```

不允许客户端把 LocalStorage 旧状态和新 snapshot 双向合并。

### 8.9 去重、乱序与 ACK

- 相同 `event_id` 只应用一次。
- 同一 `stream + stream_id` 小于当前序号的 `seq` 直接忽略。
- 大于当前序号 + 1 视为缺口，停止应用并请求快照。
- `room`、`room_chat` 和 `game` 序号独立，不能交叉比较。
- Redis 重投、WebSocket 重发仍使用同一 `event_id`。
- 权威事件不依赖客户端 ACK；ACK 只用于显式声明需要 ACK 的非权威流类型。
- 聊天消息以 `client_message_id + sender_scope` 做 REST/WS 去重。

### 8.10 AI 流与进度

- AI DM/NPC 文本流使用 `ai_stream`，不推进 `game_events.seq`。
- 流式片段在最终安全检查和 Go 接受前不得作为正式角色秘密或游戏事实。
- 任务生成进度使用 `ai_progress`，以 `generation_job_id` 和 stage 为作用域。
- AI 流失败不撤销已提交游戏状态；最终消息以 `npc.dialogue_generated` 或 `game.message_received` 作为正式事件。
- AI 进度事件可丢弃、可重放，不改变 `agent_runs` 状态。
- 降级时发送 `ai.run.degraded`，用户可见提示不得含内部模型/供应商错误。

### 8.11 背压和连接限制

- 每连接消息队列、帧大小和发送速率有上限。
- 慢消费者可丢弃可重放的 AI 流片段，不丢弃权威 game event。
- 队列超限时断开并让客户端使用 snapshot 恢复。
- 大 snapshot 使用 HTTP/分片获取，不进单条 WS 消息。
- Chat 和 AI 发送按用户、房间、角色限流。
- 单 IP/账号连接数、单房间连接数和帧大小可配置。

### 8.12 WebSocket 错误

`server.error` 示例：

```json
{
  "event_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "event_type": "server.error",
  "stream": "system",
  "stream_id": "01J9Z3K4M7Q2R8T5V6W1X20000",
  "seq": 2,
  "schema_version": 1,
  "game_id": "01J9Z3K4M7Q2R8T5V6W1X30000",
  "timestamp": "2026-11-21T10:07:00Z",
  "payload": {
    "code": "FORBIDDEN_VIEW",
    "message": "当前连接无权读取该游戏通道",
    "retryable": false,
    "request_id": "01J9Z3K4M7Q2R8T5V6W1X40000"
  }
}
```

连接级协议错误使用 WebSocket close code：

| Close code | 含义 |
| --- | --- |
| `1008` | 策略、Origin 或协议违规 |
| `4001` | Ticket/Session 认证失败 |
| `4002` | Ticket 过期或已使用 |
| `4003` | 房间访问被拒绝 |
| `4004` | 游戏访问被拒绝 |
| `4008` | WebSocket 协议版本不支持 |
| `4009` | Session 被撤销/账号受限 |
| `4010` | 连接超过并发上限 |

---

## 9. 统一错误码

### 9.1 错误码规范

- `code` 为稳定大写 snake_case。
- 客户端按 `code` 分支，不解析 `message`。
- `details` 只放安全、有限、结构化信息。
- `retryable=true` 只表示可使用相同幂等键或重新获取快照重试。
- `INTERNAL_ERROR` 不返回堆栈、SQL、Prompt、模型供应商原始错误。

### 9.2 公共与认证错误

| HTTP | Code | 场景 | Retryable |
| --- | --- | --- | --- |
| 400 | `VALIDATION_ERROR` | 参数缺失、类型或格式错误 | false |
| 400 | `INVALID_CURSOR` | 游标非法或已失效 | false |
| 400 | `INVALID_SEQ_RANGE` | 事件序号范围非法 | false |
| 401 | `AUTH_REQUIRED` | 未登录或 Session 缺失 | false |
| 401 | `AUTH_INVALID` | 登录凭证错误 | false |
| 403 | `ACCOUNT_RESTRICTED` | 账号封禁、停用或注销中 | false |
| 403 | `CSRF_INVALID` | CSRF/Origin 校验失败 | false |
| 403 | `SESSION_REVOKED` | Session 已撤销 | false |
| 404 | `USER_NOT_FOUND` | 用户不存在或不可见 | false |
| 409 | `IDEMPOTENCY_CONFLICT` | 同幂等键不同 payload | false |
| 409 | `VERSION_CONFLICT` | 编辑器 row_version/If-Match 冲突 | true |
| 429 | `RATE_LIMITED` | 请求过频 | true |
| 500 | `INTERNAL_ERROR` | 未分类服务错误 | true |
| 503 | `DEPENDENCY_ERROR` | 依赖暂时不可用 | true |

### 9.3 权限错误

| HTTP | Code | 场景 |
| --- | --- | --- |
| 403 | `FORBIDDEN_VIEW` | 无权读取对象或字段 |
| 403 | `FORBIDDEN_ACTION` | 有权查看但无权操作 |
| 403 | `FORBIDDEN_EDIT` | 非 owner/协作编辑者 |
| 403 | `NOT_ROOM_MEMBER` | 不是有效房间成员 |
| 403 | `NOT_GAME_PLAYER` | 不是有效游戏玩家 |
| 403 | `ROLE_NOT_BOUND` | 玩家没有有效角色/身份绑定 |
| 403 | `HOST_REQUIRED` | 需要房主权限 |
| 403 | `ADMIN_APPROVAL_REQUIRED` | 需要审批的敏感访问 |
| 403 | `AUDIT_APPROVAL_REQUIRED` | 需要审计审批 |

### 9.4 用户、好友和内容错误

| HTTP | Code | 场景 | Retryable |
| --- | --- | --- | --- |
| 400 | `VERIFICATION_CODE_INVALID` | 验证码错误 | false |
| 400 | `VERIFICATION_CODE_EXPIRED` | 验证码过期 | true |
| 409 | `IDENTIFIER_ALREADY_EXISTS` | 认证标识已绑定账号 | false |
| 409 | `QUOTA_EXCEEDED` | 用户、创建或调用配额不足 | false |
| 400 | `NICKNAME_CONFLICT` | 昵称/资料冲突 | false |
| 400 | `TERMS_NOT_ACCEPTED` | 协议未确认 | false |
| 403 | `SELF_ACTION_FORBIDDEN` | 对本人发好友/拉黑等 | false |
| 403 | `FRIEND_REQUEST_FORBIDDEN` | 隐私或对方设置禁止 | false |
| 403 | `USER_BLOCKED` | 拉黑关系阻止操作 | false |
| 404 | `FRIENDSHIP_NOT_FOUND` | 关系不存在 | false |
| 409 | `FRIENDSHIP_EXISTS` | 已有关系 | false |
| 404 | `CONTENT_NOT_FOUND` | 剧本/版本不存在 | false |
| 404 | `SCRIPT_NOT_FOUND` | 剧本项目不存在或不可见 | false |
| 404 | `CONTENT_VERSION_UNAVAILABLE` | 指定内容版本不可用 | false |
| 409 | `CONTENT_UNAVAILABLE` | 下架、审核中或不可开局 | false |
| 400 | `UNSUPPORTED_FILE_TYPE` | 文件格式不支持 | false |
| 413 | `FILE_TOO_LARGE` | 文件超过配额 | false |
| 400 | `RIGHTS_DECLARATION_REQUIRED` | 未声明版权 | false |
| 404 | `DOCUMENT_NOT_FOUND` | 文档不存在 | false |
| 404 | `UPLOAD_NOT_FOUND` | 上传对象或上传意图不存在 | false |
| 409 | `DOCUMENT_NOT_READY` | 扫描/解析未完成 | true |
| 422 | `FILE_HASH_MISMATCH` | 上传后哈希不一致 | false |
| 422 | `FILE_REJECTED` | 扫描或内容类型拒绝 | false |
| 422 | `SCHEMA_INVALID` | 内容 Schema 非法 | false |
| 409 | `SCHEMA_UNSUPPORTED` | Schema Major 不兼容 | false |
| 422 | `VALIDATION_TIMEOUT` | 校验超时 | true |
| 409 | `PUBLISH_BLOCKED` | 存在阻断校验问题 | false |
| 409 | `VALIDATION_STALE` | 使用过期校验报告发布 | true |
| 409 | `MODERATION_REQUIRED` | 公开发布未审核 | false |
| 409 | `MODERATION_REJECTED` | 审核拒绝 | false |
| 409 | `VERSION_FROZEN` | 已发布版本不可编辑 | false |
| 409 | `VERSION_NOT_FOUND` | 版本不存在 | false |
| 404 | `PUBLICATION_NOT_FOUND` | 发布记录不存在 | false |
| 404 | `ROLE_NOT_FOUND` | 角色内容不存在 | false |
| 404 | `PHASE_NOT_FOUND` | 阶段内容不存在 | false |
| 400 | `CONTENT_LENGTH_INVALID` | 文本、评论或发言长度不符合约束 | false |
| 404 | `OBJECT_NOT_FOUND` | 编辑器对象不存在 | false |
| 400 | `OBJECT_TYPE_MISMATCH` | 对象类型与 ID 不匹配 | false |
| 400 | `UNSUPPORTED_OBJECT_TYPE` | 不在对象白名单 | false |
| 409 | `LOCKED_FIELD` | 字段被锁定禁止 Repair | false |
| 401 | `REQUEST_SIGNATURE_INVALID` | 内部请求签名缺失或不匹配 | false |
| 401 | `REQUEST_TIMESTAMP_INVALID` | 内部请求时间戳超出窗口 | false |
| 401 | `REQUEST_REPLAYED` | 内部请求 nonce 已使用 | false |
| 403 | `TOOL_TRUST_BLOCKED` | 工具来源/信任等级未获批准 | false |
| 403 | `TOOL_CONFIRMATION_REQUIRED` | 高风险工具缺少人工确认 | false |
| 403 | `SENSITIVE_CONTEXT_BLOCKED` | 受限上下文不允许发送模型 | false |
| 422 | `OUTPUT_SCHEMA_NOT_FOUND` | 任务级输出 Schema 未注册 | false |
| 422 | `OUTPUT_SCHEMA_VERSION_UNSUPPORTED` | 输出 Schema 版本不支持 | false |
| 404 | `RUN_NOT_FOUND` | Agent Run 不存在 | false |
| 409 | `CHECKPOINT_UNAVAILABLE` | Checkpoint 不存在或不兼容 | true |

### 9.5 生成与 AI 错误

| HTTP | Code | 场景 | Retryable |
| --- | --- | --- | --- |
| 422 | `PARSE_FAILED` | 文档不可读/解析失败 | true |
| 409 | `GENERATION_BLOCKED` | 生成被阻断问题阻塞 | false |
| 409 | `GENERATION_QUOTA_EXCEEDED` | 生成任务配额耗尽 | false |
| 404 | `JOB_NOT_FOUND` | 生成任务不存在 | false |
| 409 | `JOB_INVALID_STATE` | 当前任务状态不允许 action | false |
| 502 | `WORKFLOW_FAILED` | LangGraph 工作流失败 | true |
| 503 | `STREAM_UNAVAILABLE` | SSE/WebSocket AI 流不可用 | true |
| 504 | `AI_TIMEOUT` | DM/NPC/解析超时 | true |
| 409 | `AI_SAFETY_BLOCKED` | 输入/输出安全拦截 | false |
| 409 | `AI_RUN_NOT_FOUND` | Run 不存在或不可见 | false |
| 503 | `PROVIDER_UNAVAILABLE` | 模型供应商不可用或熔断 | true |
| 409 | `HINT_QUOTA_EXHAUSTED` | DM 提示配额耗尽 | false |
| 409 | `TARGET_NOT_FOUND` | 重生成目标不存在 | false |

### 9.6 房间错误

| HTTP | Code | 场景 | Retryable |
| --- | --- | --- | --- |
| 404 | `ROOM_NOT_FOUND` | 房间不存在或不可见 | false |
| 409 | `ROOM_FULL` | 无席位 | false |
| 409 | `ALREADY_JOINED` | 已是有效成员 | false |
| 409 | `ROOM_NOT_WAITING` | 房间已开局/关闭 | false |
| 409 | `ROOM_CLOSED` | 房间已关闭或过期 | false |
| 409 | `PLAYERS_NOT_READY` | 开局有人未准备 | false |
| 409 | `PLAYER_COUNT_MISMATCH` | 人数不满足内容配置 | false |
| 409 | `GAME_ALREADY_STARTED` | 重复开始 | false |
| 403 | `PASSWORD_INVALID` | 房间密码错误 | false |
| 410 | `INVITATION_INVALID` | 邀请无效 | false |
| 410 | `INVITATION_EXPIRED` | 邀请过期/撤销 | false |
| 409 | `PAUSE_NOT_ALLOWED` | 当前模式/阶段不可暂停 | false |
| 409 | `ROOM_IN_GAME` | 开局后不允许该房间操作 | false |
| 403 | `INVITATION_CREATE_FORBIDDEN` | 当前成员或房间状态不允许创建邀请 | false |
| 409 | `HOST_TRANSFER_REQUIRED` | 房主操作需要先转移 | false |
| 409 | `TARGET_NOT_MEMBER` | 控制目标不在房间 | false |

### 9.7 游戏与模式错误

| HTTP | Code | 场景 | Retryable |
| --- | --- | --- | --- |
| 404 | `GAME_NOT_FOUND` | 对局不存在或不可见 | false |
| 409 | `GAME_NOT_STARTED` | 游戏尚未开始 | false |
| 409 | `GAME_ALREADY_ENDED` | 游戏已结束 | false |
| 409 | `STATE_VERSION_STALE` | 客户端版本过旧 | true |
| 409 | `STATE_VERSION_CONFLICT` | 写入版本不匹配 | true |
| 409 | `PHASE_NOT_ALLOWED` | 当前阶段不允许动作 | false |
| 409 | `ACTION_NOT_ALLOWED` | 意图类型/参数不合法 | false |
| 409 | `TARGET_INVALID` | 目标玩家/角色/单位非法 | false |
| 409 | `NO_VOTING_RIGHT` | 无投票权 | false |
| 409 | `VOTE_CLOSED` | 投票窗口关闭 | false |
| 409 | `ROUND_NOT_FOUND` | 票型轮次不存在 | false |
| 409 | `SEARCH_QUOTA_EXHAUSTED` | 搜证次数用尽 | false |
| 409 | `SEARCH_POINT_UNAVAILABLE` | 搜证点不可用 | false |
| 409 | `CLUE_ALREADY_CLAIMED` | 独占线索已被领取 | false |
| 409 | `QUESTION_LIMIT_EXCEEDED` | 质询次数超限 | false |
| 409 | `NOT_YOUR_TURN` | 不是当前发言者 | false |
| 409 | `PLAYER_ELIMINATED` | 已淘汰玩家操作 | false |
| 409 | `LAST_WORDS_NOT_ALLOWED` | 当前状态不能遗言 | false |
| 409 | `LAST_WORDS_ALREADY_SUBMITTED` | 遗言已提交 | false |
| 409 | `DEATH_EVENT_MISMATCH` | 遗言或技能触发对应的死亡事件不匹配 | false |
| 409 | `SPEECH_TOO_LONG` | 发言超过当前阶段长度限制 | false |
| 409 | `NO_ACTIVE_SKILL` | 当前没有可用技能 | false |
| 409 | `SKILL_ALREADY_USED` | 技能已用 | false |
| 409 | `SKILL_NOT_AVAILABLE` | 技能不可触发 | false |
| 409 | `SKILL_TRIGGER_INVALID` | 触发事件不匹配 | false |
| 409 | `SKILL_CHAIN_CONFLICT` | 技能链状态冲突 | false |
| 409 | `NIGHT_ACTION_WINDOW_CLOSED` | 夜间行动窗口关闭 | false |
| 409 | `MESSAGE_TOO_LONG` | 消息超长 | false |
| 409 | `MUTED` | 当前被禁言 | false |
| 409 | `TARGET_NOT_IN_ROOM` | 消息目标不在房间 | false |
| 409 | `TARGET_NOT_IN_SCENE` | NPC/玩家不在场景 | false |
| 409 | `NPC_NOT_FOUND` | NPC 不存在 | false |
| 409 | `NPC_NOT_PRESENT` | NPC 不在场 | false |
| 409 | `DIALOGUE_WINDOW_CLOSED` | 当前对话窗口或排队条件关闭 | false |
| 409 | `SCENE_STATE_UNAVAILABLE` | 场景状态不可读 | true |
| 409 | `INVALID_SCENE` | 场景 ID 或当前场景上下文非法 | false |
| 409 | `TARGET_UNREACHABLE` | 移动目标不可达 | false |
| 409 | `RESOURCE_NOT_ENOUGH` | 资源不足 | false |
| 409 | `SCENE_VERSION_CONFLICT` | 场景版本冲突 | true |
| 409 | `ACTION_CONFLICT` | 世界资源/行动冲突 | false |
| 409 | `EXPLORE_QUOTA_EXHAUSTED` | 探索次数耗尽 | false |
| 404 | `TASK_NOT_FOUND` | 任务不存在/不可见 | false |
| 409 | `TASK_NOT_AVAILABLE` | 任务不可接 | false |
| 409 | `TASK_PREREQUISITE_MISSING` | 前置条件不满足 | false |
| 409 | `ITEM_NOT_FOUND` | 物品不存在 | false |
| 409 | `ITEM_NOT_OWNED` | 物品不属于玩家/队伍 | false |
| 409 | `ITEM_EFFECT_UNAVAILABLE` | 物品效果不可用 | false |
| 409 | `CHOICE_NOT_AVAILABLE` | 剧情选择不可用 | false |
| 409 | `PARTY_CONFIRMATION_REQUIRED` | 多人确认未完成 | false |
| 409 | `ENDING_NOT_AVAILABLE` | 条件未满足 | false |

### 9.8 结果、社区和实时错误

| HTTP | Code | 场景 | Retryable |
| --- | --- | --- | --- |
| 409 | `RESULT_NOT_FOUND` | 结算尚未生成 | true |
| 409 | `RESULT_INCOMPLETE` | 结算未完成 | true |
| 409 | `REPLAY_NOT_READY` | 复盘生成中 | true |
| 409 | `REPLAY_VERSION_UNSUPPORTED` | 旧 replay/package 不支持 | false |
| 403 | `REVIEW_NOT_ALLOWED` | 未完成对局 | false |
| 409 | `GAME_NOT_COMPLETED` | 对局未完成，不能评分 | false |
| 409 | `REVIEW_ALREADY_EXISTS` | 同用户同版本已有评分 | false |
| 409 | `COMMENT_TOO_LONG` | 评论超过长度限制 | false |
| 409 | `OBJECT_NOT_SHAREABLE` | 对象不允许分享 | false |
| 409 | `DUPLICATE_REPORT` | 重复举报 | false |
| 400 | `REASON_INVALID` | 举报原因非法 | false |
| 409 | `FEATURE_DISABLED` | 社区功能未开放 | false |
| 409 | `MODERATION_PENDING` | 内容待审核 | true |
| 400 | `PROTOCOL_UNSUPPORTED` | 客户端/WS 协议不支持 | false |
| 409 | `EVENT_GAP_TOO_LARGE` | 增量过大，必须 snapshot | true |
| 410 | `INVALID_MESSAGE_RANGE` | 消息游标范围失效 | false |
| 503 | `AUTH_UPSTREAM_UNAVAILABLE` | 认证依赖不可用 | true |
| 503 | `SERVICE_UNAVAILABLE` | 服务维护、容量保护或依赖全面不可用 | true |

### 9.9 错误展示原则

- 客户端根据 `code` 显示稳定文案，`message` 只作为开发/调试补充。
- 多字段校验使用 `details.field_errors`，例如 `{"field_errors":[{"field":"rating","code":"OUT_OF_RANGE"}]}`。
- 版本冲突返回 `details.expected_version/current_version`。
- 房间已关闭返回 `details.room_status`，引导用户去结果/历史/大厅。
- AI 安全拦截不返回被拦截原文、命中实体名称或内部 Prompt。
- 错误响应不得包含数据库字段、SQL 语句、堆栈、模型供应商原始响应。
- `IDEMPOTENT_REPLAY` 不是错误码；幂等成功必须读取 `meta.idempotent_replay=true`。
- `REALTIME_DISCONNECTED` 是客户端连接状态，不是服务端 HTTP 错误码。

---

## 10. 幂等、并发和重复请求

### 10.1 幂等范围

| 场景 | 幂等键作用域 | 存储/约束 |
| --- | --- | --- |
| 游戏意图 | `game_id + idempotency_key` | `game_intents` 唯一键 |
| 房间创建/加入/开始 | `room_scope + idempotency_key` | room/member/session 唯一约束 |
| 生成任务 | `script_id + job_type + idempotency_key` | `generation_jobs` 唯一键 |
| 聊天消息 | `room_id + sender + client_message_id` | `game_messages` 唯一键 |
| 投票 | `game_id + round_key + voter + ballot_type` | `game_votes` 唯一键 |
| 搜索 | `game_id + idempotency_key` | `game_intents` + claim 唯一键 |
| 发布 | `script_version_id + channel + idempotency_key` | publication 唯一键 |
| AI Run | `task_type + idempotency_scope + idempotency_key` | `agent_runs` 唯一键 |

普通幂等结果建议至少保留 7 天；游戏意图、结算、发布和计费相关结果按数据库策略长期保留。

### 10.2 幂等处理

1. 服务端规范化请求 payload。
2. 计算 payload hash。
3. 查找作用域内幂等记录。
4. 同键同 hash 返回原状态码、原资源/结果和 `meta.idempotent_replay=true`。
5. 同键不同 hash 返回 `409 IDEMPOTENCY_CONFLICT`。
6. 首次执行成功后在业务事务内写结果。
7. HTTP 重试、WebSocket 重发、队列重投共用同一逻辑。

未提交成功的请求不得提前写成功幂等结果。失败结果可选择短时缓存，但不得吞掉后续合法版本重试。

### 10.3 游戏并发

游戏写操作必须包含：

- `expected_version`
- `Idempotency-Key`
- `client_request_id` 可选

服务端事务：

```text
认证/成员/角色
→ 幂等查询
→ 锁定 game_sessions
→ 校验 expected_version
→ 规则引擎 Validate + Apply
→ 写 game_events / 模式专用表
→ 更新 game_states / state_version / state_hash
→ 写 game_intents
→ 写 outbox
→ commit
→ WS 投影广播
```

常见冲突：

| 场景 | 返回 |
| --- | --- |
| `expected_version` 小于当前 | `409 STATE_VERSION_STALE` |
| 并发同时推进同一版本 | 一个成功，其他 `409 STATE_VERSION_CONFLICT` |
| 同一投票重复提交 | 幂等回放原结果 |
| 改票使用新幂等键但版本正确 | 规则允许时更新有效票 |
| 独占线索争抢 | 一个成功，其他 `409 CLUE_ALREADY_CLAIMED` |
| 技能链重复触发 | 同一事件链返回冲突或原结果 |
| 结算请求重复 | 返回已有 result，不插入第二条 |

### 10.4 编辑器并发

- 编辑器请求带 `row_version` 或 `If-Match`。
- 保存冲突返回 `409 VERSION_CONFLICT` 和当前摘要，不静默覆盖。
- 客户端必须提供“重新加载”或“保留本地差异”选择。
- 局部重生成以 `candidate_id` 和锁定字段运行，不原地覆盖已接受对象。
- 发布使用 `validation_id`、`row_version` 和原子指针切换。

### 10.5 房间并发

- 加入房间使用房间行锁和席位/有效成员唯一约束。
- 最后一席只允许一个事务成功。
- 准备状态更新校验 `expected_room_event_seq`。
- 开局锁定 room，校验人数、准备、内容版本和规则版本，原子创建 GameSession。
- 重复 start 不创建第二个 GameSession。
- 房主迁移使用 `room_members`/`game_rooms` 条件和幂等命令。

### 10.6 客户端重试规则

- `retryable=false`：不要自动重试。
- `retryable=true`：先用相同 `Idempotency-Key` 重试。
- 版本冲突：先刷新状态/快照，再决定是否用新幂等键重新提交。
- `AI_TIMEOUT`：可切换模板/降级，不重复执行已提交游戏动作。
- `EVENT_GAP_TOO_LARGE`：停止应用增量，执行完整 resync。
- 前端不得用乐观状态提前显示胜负、资源扣除或私密身份。

---

## 11. API 版本管理与 OpenAPI 组织

### 11.1 兼容性规则

| 变更 | 兼容性 |
| --- | --- |
| 新增可选响应字段 | 兼容 |
| 新增枚举值且客户端必须忽略未知值 | 兼容，需声明 |
| 新增可选请求字段 | 兼容 |
| 新增错误码 | 兼容 |
| 改变字段语义/默认值 | 不兼容 |
| 删除/重命名字段 | 不兼容 |
| 改变鉴权/权限规则 | 不兼容或需版本化配置 |
| WebSocket 事件 payload 不兼容 | 新 major 子协议 |
| 新增游戏模式专属路径 | 兼容 |
| 将模式路由合并成通用万能动作 | 不采用 |

### 11.2 弃用流程

1. OpenAPI 标记 `deprecated: true`。
2. 响应增加 `Sunset` 或 `Deprecation` header。
3. 至少保留一个客户端发布周期。
4. 监控旧客户端调用量。
5. 迁移完成后移除旧 endpoint。
6. GamePackage 和进行中对局固定原 API/引擎/package 版本。

### 11.3 OpenAPI 目录建议

```text
packages/contracts/openapi/
├── openapi.yaml
├── paths/
│   ├── auth.yaml
│   ├── users.yaml
│   ├── friends.yaml
│   ├── scripts.yaml
│   ├── generation-jobs.yaml
│   ├── rooms.yaml
│   ├── realtime.yaml
│   ├── games-common.yaml
│   ├── games-murder.yaml
│   ├── games-werewolf.yaml
│   ├── games-rpg.yaml
│   ├── ai-runs.yaml
│   └── community.yaml
├── components/
│   ├── schemas/
│   │   ├── common.yaml
│   │   ├── auth.yaml
│   │   ├── script.yaml
│   │   ├── room.yaml
│   │   ├── game-projection.yaml
│   │   ├── game-events.yaml
│   │   ├── game-messages.yaml
│   │   ├── ai.yaml
│   │   └── community.yaml
│   ├── parameters/
│   ├── responses/
│   ├── security/
│   └── examples/
└── README.md
```

WebSocket、SSE 和内部事件 Schema 放在：

```text
packages/contracts/json-schema/
├── ws-messages/
├── sse-events/
├── game-events/
└── ai-outputs/
```

### 11.4 OpenAPI 规范要求

- 每个 operation 必须有唯一 `operationId`，采用 `auth_register`、`scripts_create`、`rooms_join`、`games_murder_search` 形式。
- `security` 明确 `cookieAuth`、`csrfHeader` 和匿名访问。
- 每个 DTO 字段带 `description`、`example`、`nullable`、枚举和长度约束。
- 私密字段使用 `x-sensitivity: D2|D3|D4`。
- 幂等写操作使用 `x-idempotency-required: true`。
- `429`、`401`、`403`、`409`、`422`、`500`、`503` 使用统一错误响应组件。
- 游标分页使用统一 `CursorPage`。
- 时间字段统一 `format: date-time`。
- 所有金额使用 decimal string 或明确 JSON number 约束，不使用浮点隐式舍入。
- `additionalProperties: false` 用于请求和高风险响应 Schema。
- 错误 `details` 使用按错误类型定义的 Schema，不使用任意对象长期漂移。

### 11.5 代码生成

生成目标：

```text
apps/web/src/generated/api/
backend/internal/contracts/
ai_service/src/mysteryverse_ai/schemas/generated/
```

CI 要求：

- OpenAPI lint。
- JSON Schema lint。
- Go/TS/Python 生成代码无漂移。
- 所有示例可反序列化和 Schema 校验。
- 路由与 operationId 唯一性检查。
- DTO 敏感字段泄漏测试。
- WebSocket fixture 双端合同测试。
- Breaking change 检查。

### 11.6 自动泄漏检查

禁止生成类型中出现以下未授权公开字段名：

- `secret_payload_id`
- `ciphertext`
- `nonce`
- `key_id`
- `private_payload`
- `role_assignment_table`
- `npc_memory_raw`
- `system_prompt`
- `chain_of_thought`

允许出现在内部契约或审计契约中的字段必须显式白名单并绑定独立 Schema。

---

## 12. 验收与实施检查

### 12.1 API 契约验收

- 每个 REST operation 都在 OpenAPI 中存在，method、status、错误响应和幂等要求一致。
- 每个写接口都有 `Idempotency-Key` 规则。
- 每个游戏动作都有 `expected_version`。
- 所有私密接口都有 viewer/role/phase 权限校验。
- 房间、游戏、生成任务和发布操作都有并发/重复请求测试。
- WebSocket 公共/私密事件均有投递测试。
- 事件缺口、旧序号、重复事件和大快照恢复有端到端测试。
- AI 超时、安全拦截、模板降级和消息去重有测试。
- 错误码稳定，客户端不依赖自然语言错误消息。

### 12.2 安全验收

- URL/body 修改 `script_id/user_id/game_id/room_id` 无法越权。
- 房主、作者、普通运营无法读取其他角色本、身份表或私聊。
- 狼人 AI/好人玩家无法读到狼队频道或完整身份分配。
- NPC 对话不能通过注入获取未授权记忆或真相。
- 公共 WS 频道不出现私人线索、角色秘密、AI 内部上下文。
- share token、导图、复盘、搜索和对象存储旁路全部测试。
- 账号退出后旧 Session、旧 ticket 和旧 WebSocket 不能继续写。

### 12.3 性能验收

- 游戏命令 REST 确认延迟不包含模型调用，满足 TECH_DESIGN 的 P95。
- WS 从数据库提交到客户端投影满足目标延迟。
- 重连 snapshot + delta 在目标时间内可操作。
- 房间最后一席、线索争抢、重复投票和结算重复请求有并发测试。
- SSE/WS AI 流不阻塞权威游戏事件；慢客户端不影响其他连接。
- 生成任务进度只发送摘要，不把完整模型上下文塞入消息。

### 12.4 文档边界

本文定义公共前后端 API、WebSocket 协议和契约组织建议，不包含：

- API Handler 或框架业务代码。
- Python 内部 Prompt 与 Agent 实现。
- 数据库 DDL 迁移。
- 后台审核 API 的完整运营权限流程。
- 语音、支付、推荐和原生 App API。

Go 与 Python 的内部接口仍由独立内部 OpenAPI 管理，必须使用 mTLS/短期服务 Token、请求签名、`trace_id`、`run_id` 和幂等键，且不通过浏览器网络暴露。
