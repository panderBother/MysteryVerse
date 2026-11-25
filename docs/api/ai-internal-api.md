# 内部 AI API

> 状态：DQ-19 内部安全基础 APPROVED / IMPLEMENTED / TEST-VERIFIED
> 网络：不暴露公网
> 依据：ADR-001、ADR-004、ADR-007、ADR-008、ADR-009

## 1. 认证与通用字段

生产内部请求必须同时使用 mTLS 与 HMAC-SHA256。开发/测试可使用显式配置的
短期 Service Token，但不得把它当作完整生产认证。请求必须携带：

- `X-Service-Id`
- `X-Service-Token`
- `X-Service-Scope`
- `X-Service-Tenant`（有租户上下文时）
- `X-Request-Id`
- `X-Trace-Id`
- `Idempotency-Key`
- `run_id` 或 `job_id`
- `schema_version`

配置签名后还必须携带：

```text
X-Request-Signature-Version: mv-hmac-sha256-v1
X-Service-Key-Id: <key-id>
X-Request-Timestamp: <unix-seconds>
X-Request-Nonce: <8-128 URL-safe characters>
X-Request-Signature: <lowercase hex HMAC-SHA256>
```

Canonical 字符串按 UTF-8 和以下顺序以 `\n` 连接：

```text
signature_version
key_id
timestamp
nonce
service_id
HTTP_METHOD
request_path
raw_query
hex(sha256(request_body))
```

签名使用 Key ID 对应的密钥，并采用恒定时间比较。Timestamp 默认窗口 300 秒；
Nonce 在至少双倍窗口内原子消费一次。Key ID 未知、撤销、KMS 不可用或 Nonce
Store 不可用时均 fail-closed。

Scope 格式为逗号分隔的 allowlist：

```text
ai:agent:run
run:<run_id>
tenant:<tenant_id>
```

认证成功不等于授权成功。服务必须同时匹配 operation scope、目标 resource scope
以及适用的 tenant scope。客户端提交的 Tenant ID 只有在可信 Service Identity
已经被认证、且 scope 中包含对应 `tenant:<id>` 时才生效。

## 2. 同步 AI 回合

```text
POST /internal/v1/ai/dm-turn
POST /internal/v1/ai/npc-turn
POST /internal/v1/ai/parse-action
POST /internal/v1/ai/explain-rule
GET  /internal/v1/ai/runs/{run_id}
```

同步请求由 Go 提供已授权 `AllowedKnowledgeSlice`、actor、phase、
`game_id`、`state_version`、`event_seq`、预算和输出约束。

同步响应：

```json
{
  "run_id": "01J9Z3K4M7Q2R8T5V6W1X10000",
  "schema_version": 1,
  "output": {
    "speech": "你们的时间线仍有一处矛盾。",
    "intent": null,
    "referenced_event_ids": ["01J9Z3K4M7Q2R8T5V6W1X20000"],
    "visibility": "public"
  },
  "safety_flags": [],
  "usage": {
    "input_tokens": 1234,
    "output_tokens": 180
  }
}
```

同一同步回合内，Python 不得回调 Go Context Resolve 再触发新的 AI 调用。
Go 必须在调用前完成全部上下文组装。

### 2.1 Task 8 通用 Agent Runtime

```text
POST /internal/v1/ai/agent-runs
```

当前实现接受 Contracts `AgentRequest` 的兼容扩展字段：

- `task_type`、`task_goal`。
- `context`：Go 已授权的任务上下文、游戏投影、近期消息、摘要和偏好。
- `allowed_tools`、`allowed_permissions`。
- `execution_limits`：`timeout_ms`、`max_steps`、`max_context_tokens`、
  `max_output_tokens`。
- `trace_id`。

请求头必须包含 `X-Service-Id`、`X-Service-Token`、`X-Service-Scope`、
`X-Request-Id`、`X-Trace-Id` 和 `Idempotency-Key`。当前 endpoint 要求
`ai:agent:run` 与 `run:<run_id>`。

Task 11 正式安全 profile 在配置签名密钥后额外要求：

```text
X-Request-Signature-Version
X-Service-Key-Id
X-Request-Timestamp
X-Request-Nonce
X-Request-Signature
```

签名算法固定为 `mv-hmac-sha256-v1`。Key ID 支持 Active/Previous 轮换和撤销；
开发环境使用环境变量 Key Provider，生产配置要求外部 KMS。多实例生产配置要求
Redis nonce backend，Store 故障返回 `NONCE_STORE_UNAVAILABLE`，不会放行。

响应返回 Contracts `AgentResponse`：

- `status`：`succeeded|failed|timeout|cancelled`。
- `structured_output`：经过任务级 Pydantic Schema 校验的对象。
- `output_schema_key` / `output_schema_version`：实际使用的任务级 Schema 身份。
- `validation_errors`：不含原始输入的路径、消息和错误类型。
- `tool_calls`：工具名、参数 hash、状态、耗时和稳定错误码；不返回原始参数。
- `steps`：context/model/tool/validate trace、Token、耗时和错误码。
- `usage`、`error`、`request_id`、`trace_id`。

Agent 的 `structured_output` 是候选/表达结果，不是 Go 游戏状态补丁。Go 只可读取
允许字段，并在自己的规则引擎中校验任何建议动作。Python 不得返回可直接执行的
权威状态变更。

当前 Demo 只注册 `demo` 任务和 `mock_context_lookup` 工具。其他任务类型在没有
任务级 Schema 与 Go 侧校验前返回 `VALIDATION_ERROR`。

请求可通过 `output_schema_key` / `output_schema_version` 选择已注册 Schema。
模型无权选择或修改 Schema。未注册 key 返回 `OUTPUT_SCHEMA_NOT_FOUND`，不支持
的版本返回 `OUTPUT_SCHEMA_VERSION_UNSUPPORTED`。

D3/D4 近期上下文默认不进入模型。`AI_ALLOW_RESTRICTED_MODEL_CONTEXT=true`
只作为显式高风险配置，不能在 SecretPayload AAD 和私密数据政策批准前用于生产。

### 2.2 超时、取消与幂等

- 父级 `budget_ms` 与 `execution_limits.timeout_ms` 共同限制一次执行。
- HTTP 客户端断开时 Python 设置取消事件并终止 LangGraph 任务。
- `max_steps` 同时由 Runtime 计数和 LangGraph `recursion_limit` 限制。
- 同一 `run_id + Idempotency-Key` 的同 payload 返回缓存结果；不同 payload 返回
  `IDEMPOTENCY_CONFLICT`。
- 语义 `failed|timeout|cancelled` 使用 HTTP 200 的 AgentResponse；请求协议、
  服务认证或幂等冲突使用标准 API error envelope。

## 3. 上下文与权威事实

```text
POST /internal/v1/context/resolve
POST /internal/v1/authority/facts/query
```

Go 返回：

- 授权实体字段。
- 可见事件 ID 和摘要。
- reveal gate。
- forbidden entity hashes。
- legal actions。
- context expiry。

Python 不得通过实体 ID 猜测或额外查询扩大切片。

## 4. MemoryCandidate

```text
POST /internal/v1/ai/memory-candidates
```

请求见 [ADR-001](../decisions/ADR-001-memory-authority.md)。

Go 返回稳定状态：

```json
{
  "candidate_id": "01J9Z3K4M7Q2R8T5V6W1X30000",
  "status": "accepted",
  "memory_id": "01J9Z3K4M7Q2R8T5V6W1X80000",
  "record_version": 1,
  "deduplicated": false
}
```

`rejected|duplicate|conflict` 不得被 Python 自动重试为语义成功。

## 5. Artifact 与进度

```text
POST /internal/v1/ai/artifacts/register
POST /internal/v1/ai/progress
```

Artifact 注册顺序：

1. Python 上传对象存储。
2. 计算 SHA-256。
3. 调用注册 API。
4. Go 校验 job、workflow、Schema、大小和 hash。
5. Go 写 staging 元数据。
6. Python 发送完成回调。

注册和回调必须幂等。重复 artifact 返回现有记录。

进度事件由 Go 幂等更新，必须检查 `job_id + workflow_version`。

## 6. Intent 候选

```text
POST /internal/v1/ai/intents/validate
POST /internal/v1/ai/intents/submit
```

`validate` 无副作用。`submit` 只在规则允许且 actor 有效时进入正式 Go Intent 链。
两者都必须带 actor scope、game version 和幂等键。

## 7. 错误矩阵

| Code | HTTP | 重试 |
| --- | ---: | --- |
| `VALIDATION_ERROR` | 422 | 否 |
| `FORBIDDEN_VIEW` | 403 | 否 |
| `IDEMPOTENCY_CONFLICT` | 409 | 否 |
| `VERSION_CONFLICT` | 409 | 重新拉取 |
| `AI_TIMEOUT` | 504 | 有界 |
| `AI_SAFETY_BLOCKED` | 409 | 否 |
| `PROVIDER_UNAVAILABLE` | 503 | 有界 |
| `WORKFLOW_FAILED` | 502 | 有限 |
| `DEPENDENCY_ERROR` | 503 | 有界 |
| `REQUEST_REPLAYED` | 401 | 否 |
| `SERVICE_KEY_INVALID` | 401 | 否 |
| `SERVICE_KEY_UNAVAILABLE` | 503 | 有界 |
| `NONCE_STORE_UNAVAILABLE` | 503 | 有界 |
| `SERVICE_SCOPE_DENIED` | 403 | 否 |
| `SERVICE_RESOURCE_DENIED` | 403 | 否 |
| `SERVICE_TENANT_DENIED` | 403 | 否 |

## 8. 防环与预算

- trace 记录调用方向和深度。
- 设置最大回调深度。
- 同步回合禁止 Python → Go → Python 嵌套。
- 父级 deadline 扣除已用时间后再传给子调用。
- Provider 请求、节点、Repair 和工作流使用各自预算上限。

## 9. 测试

- 内部 API 不出公网。
- 服务 Token 受众/操作绑定。
- 重放、body hash 和 nonce。
- Key ID 撤销、双密钥轮换、KMS 和 Nonce Store 故障。
- operation/resource/tenant scope、跨租户拒绝和审计脱敏。
- mTLS 有效证书、缺失客户端证书和证书链错误。
- 同步回合防环。
- MemoryCandidate 权限和幂等。
- Artifact 重复注册。
- 错误映射和有限重试。

真实 CA、KMS 和 Redis 多实例部署当前为 `NOT VERIFIED`。测试通过不能替代生产
基础设施验证。

