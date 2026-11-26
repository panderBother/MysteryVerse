# MysteryVerse

> **AI 不是主持人外壳，而是可验证的游戏世界。**
>
> MysteryVerse（谜境）是一个 AI Native 多人互动推理游戏平台，连接 AI 剧本杀、
> AI 狼人杀与沉浸式互动 RPG。Go 负责确定性规则和权威状态，Python 负责可控的
> AI 叙事、推理与内容工作流，浏览器只消费服务端投影。

当前仓库已包含 Monorepo 工程、用户认证、等待房 REST、Go 游戏核心引擎、剧本
内容管理、版本化跨语言 Contracts Registry 和 Python Agent Runtime 安全基础。
DQ-18/DQ-19 的 Registry、HMAC、Key ID、ACL、审计和 TLS 客户端已落地并通过
测试级验证；真实 CA/KMS/Redis 多实例仍待部署。AI DM、NPC 完整玩法、外部
MCP 仍在决策门禁后。DQ-21 的 MySQL Run/Step/Checkpoint Metadata、Redis
Owner Lease、DB Fencing、AES-GCM Checkpoint、Recovery 与 Cleanup Worker
已实现并通过单元测试；真实 MySQL/Redis 多实例与 Restore 仍待基础设施验证。

## 技术栈

| 区域 | 技术 |
| --- | --- |
| Web | React、TypeScript、Vite、Tailwind CSS、React Router、TanStack Query、Zustand、Motion |
| Go | Go 1.24、Chi、`net/http`、`slog`、`sqlc`、`database/sql`、goose |
| AI | Python 3.12、FastAPI、Pydantic Settings、LangGraph、uv |
| 基础设施 | MySQL 8.0、Redis 7、go-redis |

## 目录

```text
apps/web/              React 应用
backend/               Go API、Realtime/Runtime、Worker
ai_service/            Python AI API 与 Worker
packages/contracts/    跨语言契约源
deploy/                部署、迁移与观测目录
scripts/               跨平台开发命令
tests/                 跨语言测试目录
docs/                  产品、架构、API、数据库和任务文档
```

`TECH_DESIGN.md` 使用 `apps/web`、`backend`、`ai_service`、`packages/contracts`
和 `deploy`。本仓库采用这些正式路径，不另建并行的 `frontend`、`backend/go`
或 `services/ai` 目录。

## 前置工具

- Node.js 22.18 或更高
- npm 11 或更高
- Go 1.24
- Python 3.12
- uv 0.9 或更高
- Docker Desktop/Engine，用于 MySQL 与 Redis

## 首次安装

PowerShell：

```powershell
./scripts/mv.ps1 install
Copy-Item .env.example .env
```

如果环境已安装 Make：

```text
make install
```

## 启动基础设施

```powershell
./scripts/mv.ps1 docker-up
```

停止：

```powershell
./scripts/mv.ps1 docker-down
```

MySQL 数据保存在 `mysql_data`，Redis AOF 保存在 `redis_data`。首次使用前应修改
`.env` 中的开发密码。仓库只提供示例值，不包含生产密钥。

## 数据库迁移与查询生成

迁移文件位于 `backend/migrations/`，只通过 `backend/cmd/migrate` 执行，应用
启动时不会自动修改 Schema。

```powershell
./scripts/mv.ps1 db-migrate
./scripts/mv.ps1 db-status
./scripts/mv.ps1 db-rollback
./scripts/mv.ps1 db-generate
./scripts/mv.ps1 db-check
```

`db-generate` 使用 `backend/sqlc.yaml` 从迁移定义和
`backend/internal/storage/queries/` 生成类型化 SQL 代码。生成目录
`backend/internal/storage/sqlc/` 不得手工修改。

数据库集成测试使用独立的已迁移测试库，并只接受显式测试 DSN：

```powershell
$env:MYSQL_TEST_DSN = "mv_test:migration-test@tcp(127.0.0.1:3306)/mv_repository_test?parseTime=true&loc=UTC"
$env:MYSQL_MIGRATION_TEST_DSN = "mv_test:migration-test@tcp(127.0.0.1:3306)/mv_migration_test?parseTime=true&loc=UTC"
./scripts/mv.ps1 db-test
```

迁移测试库必须为空；测试结束会完整向下回滚。Repository、Identity 和 Room
集成测试使用事务回滚或显式清理测试数据。绝不能把生产 DSN 传给这些变量。

## 启动本地应用

分别启动三个终端：

```powershell
./scripts/mv.ps1 dev
```

也可以单独启动：

```powershell
npm run dev:web
go -C backend run ./cmd/api
uv run --project ai_service python -m mysteryverse_ai.main
```

默认地址：

- Web: `http://127.0.0.1:5173`
- Go API: `http://127.0.0.1:8080`
- Python AI: `http://127.0.0.1:8000`

## 用户与房间 API

Task 5 已提供 email + 密码认证和等待房 REST 流程：

```text
POST  /api/v1/auth/register
POST  /api/v1/auth/login
POST  /api/v1/auth/logout
GET   /api/v1/me
PATCH /api/v1/me
POST  /api/v1/rooms
GET   /api/v1/rooms
GET   /api/v1/rooms/{room_id}
POST  /api/v1/rooms/{room_id}/join
POST  /api/v1/rooms/{room_id}/leave
POST  /api/v1/rooms/{room_id}/ready
POST  /api/v1/rooms/{room_id}/host-actions
GET   /api/v1/rooms/{room_id}/messages
POST  /api/v1/rooms/{room_id}/messages
GET   /api/v1/rooms/{room_id}/events
```

浏览器登录态保存在 `HttpOnly` `mv_session` Cookie；所有写请求需要
`Origin`，登录后的写请求还需要响应签发的 `X-CSRF-Token`。等待房消息使用独立
`room_messages.chat_seq`，不会修改 `room_events.seq` 或 `game_events.seq`。

前端 Mock 数据可使用
`packages/contracts/examples/auth-session.json`、
`room-projection.json`、`room-message.json` 和
`ws-room-chat-envelope.json`。

## 游戏核心引擎

Task 6 提供候选 Go 游戏运行时：

```text
POST /api/v1/rooms/{room_id}/start
GET  /api/v1/games/{game_id}/projection
GET  /api/v1/games/{game_id}/events
GET  /api/v1/games/{game_id}/result
GET  /api/v1/games/{game_id}/murder/me
POST /api/v1/games/{game_id}/murder/reading-complete
GET  /api/v1/games/{game_id}/murder/search-points
POST /api/v1/games/{game_id}/murder/searches
GET  /api/v1/games/{game_id}/murder/clues
POST /api/v1/games/{game_id}/murder/votes
GET  /api/v1/games/{game_id}/murder/votes
POST /api/v1/games/{game_id}/pause
POST /api/v1/games/{game_id}/resume
```

这些接口复用 Session/CSRF 和房间权限。ADR-004/ADR-005 尚未批准，因此事件版本与
暂停恢复仍是候选实现，不构成正式 `GATE-GAMESTATE`。

## 剧本内容管理与发布

Task 7 提供 Draft 内容、结构化对象、校验、版本克隆、私有/邀请发布和下架：

```text
POST   /api/v1/scripts
GET    /api/v1/scripts
GET    /api/v1/scripts/{script_id}
PATCH  /api/v1/scripts/{script_id}
DELETE /api/v1/scripts/{script_id}
GET    /api/v1/scripts/{script_id}/versions
POST   /api/v1/scripts/{script_id}/versions/{version_id}/drafts
GET    /api/v1/scripts/{script_id}/versions/{version_id}/editor
POST   /api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}
PATCH  /api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}
DELETE /api/v1/scripts/{script_id}/versions/{version_id}/objects/{object_type}/{object_id}
POST   /api/v1/scripts/{script_id}/versions/{version_id}/validate
POST   /api/v1/scripts/{script_id}/versions/{version_id}/publish
POST   /api/v1/publications/{publication_id}/unpublish
```

支持对象类型为 `chapter`、`stage`、`character`、`clue`、`ending`、`asset`、
`search_point` 和 `search_point_clue`。所有写请求要求 `Origin`、`X-CSRF-Token`
和 `Idempotency-Key`。发布事务会重新执行结构校验和 Go Engine 预检；阻断问题时
不会切换 `scripts.published_version_id`。

`channel=public` 在运营审核流程批准前返回 `MODERATION_REQUIRED`；
`private_profile_ref` 与 `private_text_ref` 在 SecretPayload AAD 批准前不接受写入。
阶段 entry/exit DSL 和非空结局条件返回 `unsupported` 并阻断发布。

## Python AI Agent Runtime

Task 8 在 `ai_service` 中提供通用 LangGraph Agent Runtime：

```text
POST /internal/v1/ai/agent-runs
```

默认 `AI_MODEL_PROVIDER=mock`，无需模型密钥即可完成开发和测试。真实模型使用
OpenAI-compatible `/chat/completions` Provider，在运行时通过
`AI_MODEL_API_KEY` 注入密钥。

内部请求必须携带：

```text
X-Service-Id
X-Service-Token
X-Request-Id
X-Trace-Id
Idempotency-Key
```

Agent Loop 为 `model -> tool -> model -> structured validate`，受
`execution_limits.max_steps`、timeout 和取消事件限制。工具必须预先注册并通过
`allowed_tools`、`allowed_permissions` 和 actor type 校验；模型不能执行任意本地
代码。当前只开放 `demo` 任务和 `mock_context_lookup` 工具。

Go 侧最小客户端位于 `backend/internal/ai/client`，负责服务身份、trace、
idempotency、deadline、有限重试和响应 Schema 校验。Agent 输出不是游戏状态补丁，
所有游戏状态变更仍必须由 Go 规则引擎验证和执行。

Task 9 增加了可选 HMAC 请求签名、timestamp/nonce 防重放、双密钥轮换窗口、
版本化输出 Schema、本地 Tool trust/确认和进程内 Run 元数据原型。GATE-AI 仍未
通过；逐项证据和阻断项见 `docs/architecture/gate-ai-checklist.md`。

## 健康检查

```text
GET http://127.0.0.1:8080/health/live
GET http://127.0.0.1:8080/health/ready
GET http://127.0.0.1:8080/api/v1/health
GET http://127.0.0.1:8000/health/live
GET http://127.0.0.1:8000/health/ready
```

## 统一命令

```powershell
./scripts/mv.ps1 build
./scripts/mv.ps1 test
./scripts/mv.ps1 lint
./scripts/mv.ps1 format
./scripts/mv.ps1 check
./scripts/mv.ps1 smoke
./scripts/mv.ps1 db-test
```

对应 Make 目标为 `build`、`test`、`lint`、`format`、`check`、`docker-up`、
`docker-down`、`smoke`、`db-generate`、`db-migrate`、`db-status`、
`db-check`、`db-rollback`、`db-test`。

## Contracts

`packages/contracts/json-schema/mysteryverse.schema.json` 是 Go、Python、React
共用的唯一 Schema Source。

```powershell
./scripts/contracts.ps1 generate
./scripts/contracts.ps1 validate
./scripts/contracts.ps1 check
```

生成目标为 `apps/web/src/generated/contracts.ts`、
`backend/internal/contracts/contracts_gen.go` 和
`ai_service/src/mysteryverse_ai/schemas/generated/models.py`。
详细修改流程见 `packages/contracts/README.md`。

## 工程边界

- `packages/contracts` 是跨语言契约唯一来源；生成代码禁止手改。
- Python 不直接写 `mv_core`，所有权威游戏状态由 Go 管理。
- Redis 不能作为唯一权威数据源。
- Web 只消费服务端投影，不执行权限或规则裁决。
- Proposed ADR 在负责人确认前不得冻结为实现契约。

详见 `AGENTS.md`、`docs/architecture/DECISION-BASELINE.md` 和
`docs/architecture/system-architecture.md`。
