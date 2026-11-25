# GATE-AI 准入检查清单

> 日期：2026-11-23
> 总体状态：未通过
> 原则：只有代码、测试和可复现证据才算 PASS；文档方案不算证据。

## 架构边界

| 检查项 | 状态 | 证据/阻断原因 |
| --- | --- | --- |
| Go 是游戏权威状态唯一管理者 | PASS | Python Runtime 无 Go 业务表访问；Agent 输出不是状态补丁 |
| Python 不直接修改游戏状态 | PASS | 无数据库写入和 Go mutation endpoint |
| AI 输出需由 Go 二次校验 | BLOCKED | Task 9 未接入具体 AI DM/NPC 业务；契约已有边界 |
| 跨服务协议版本管理 | PASS | Contracts 140+ 定义；Agent Schema key/version 原型 |

## 安全

| 检查项 | 状态 | 证据/阻断原因 |
| --- | --- | --- |
| Service ID + Token 基础身份 | PASS | FastAPI 依赖和 Go Client 测试 |
| HMAC 请求签名原型 | PASS | body/path/query 绑定、tamper/replay 测试；Go Client → FastAPI 跨进程签名通过 |
| mTLS | BLOCKED | DQ-19 未批准，部署环境无证书链 |
| 时间戳与 nonce 重放防护 | PASS | 单实例原型和测试；多实例存储 BLOCKED |
| 双密钥轮换窗口 | PASS | primary/previous 原型测试 |
| 正式服务 ACL scope | BLOCKED | DQ-19 未定义操作级 scope |
| 本地 Tool 白名单/权限/actor 校验 | PASS | Tool Registry 测试 |
| MCP Server 信任策略 | BLOCKED | DQ-20；外部工具当前全部拒绝 |
| SSRF/外部网络访问限制 | PASS | 当前任何 external_network 工具拒绝 |
| 工具结果大小限制/不可信标记 | PASS | 结果上限和 system 隔离指令 |
| 高风险工具人工确认 | PASS | unconfirmed 工具拒绝原型 |
| D3/D4 默认不发送给模型 | PASS | Context Builder 默认过滤和测试 |
| 私密正文日志脱敏 | PASS | 结构化日志字段脱敏和测试 |
| SecretPayload AAD/解密发送 | BLOCKED | DQ-15 未批准；未实现解密 |

## 可靠性

| 检查项 | 状态 | 证据/阻断原因 |
| --- | --- | --- |
| Agent Run 状态原型 | PASS | InMemoryAgentRunStore 生命周期测试 |
| 超时与取消 | PASS | Runtime 单元测试 |
| 最大执行步数 | PASS | LangGraph 和 Runtime 双重限制 |
| Agent API 幂等 | PASS | 进程内幂等缓存与 HTTP 测试 |
| 正式 Run 持久化 | BLOCKED | DQ-21；仅进程内原型 |
| LangGraph Checkpoint/恢复 | NOT_IMPLEMENTED | 无正式 checkpointer 和恢复端点 |
| 多实例并发隔离 | BLOCKED | DQ-21；无 owner lease |
| 故障重试与降级 | PASS | Provider 有限重试、Tool 失败、Timeout 测试 |
| Trace 关联 | PASS | request/trace/run/step 关联 |
| Trace 长期持久化 | NOT_IMPLEMENTED | 未写 `mv_ai`，未接 OpenTelemetry |

## 输出约束

| 检查项 | 状态 | 证据/阻断原因 |
| --- | --- | --- |
| 版本化任务级 Schema Registry | PASS | key/version 注册与解析测试 |
| 模型不能选择 Schema | PASS | Go 请求指定，模型只生成 output |
| 必填/枚举/范围/嵌套校验 | PASS | 通用 Pydantic Schema 原型 |
| 未知字段拒绝 | PASS | `extra=forbid` 测试 |
| 校验失败不得成功返回 | PASS | Runtime 返回 failed + validation_errors |
| 非法输出错误可追踪 | PASS | 响应 validation_errors 和 trace |
| AI DM/NPC/生成正式业务 Schema | BLOCKED | DQ-18 未批准 |

## Provider

| 检查项 | 状态 | 证据/阻断原因 |
| --- | --- | --- |
| Provider 抽象与 Mock | PASS | Provider contract tests |
| OpenAI-compatible HTTP Provider | PASS | MockTransport 实现测试 |
| 真实远程模型调用验证 | BLOCKED | 无供应商 Key/合同；不虚报 |
| API Key Secret 管理 | PASS | `SecretStr`，不落日志/仓库 |
| 模型白名单与路由策略 | NOT_IMPLEMENTED | DQ-22 |
| 限流、熔断和成本预算 | NOT_IMPLEMENTED | 仅超时/重试/Usage |
| 供应商数据政策 | BLOCKED | DQ-22 |
| Provider 故障降级模板 | NOT_IMPLEMENTED | 后续 AI DM 业务条件 |

## 测试

| 检查项 | 状态 | 证据 |
| --- | --- | --- |
| Python 单元测试 | PASS | 35 tests |
| FastAPI 集成测试 | PASS | health、agent、认证、签名、幂等 |
| Go/Python 契约测试 | PASS | 141 definitions、三端 examples |
| 真实跨进程协议测试 | PASS | Go Client → FastAPI Mock Provider，Token 与 HMAC 模式均通过 |
| 权限测试 | PASS | Tool permissions/actor/trust/confirmation |
| 认证篡改/重放测试 | PASS | signed HTTP tests |
| 超时/取消/失败测试 | PASS | Runtime tests |
| 持久化/Checkpoint 故障注入 | NOT_IMPLEMENTED | DQ-21 |
| mTLS/多实例 nonce 测试 | BLOCKED | DQ-19 |
| SSRF/真实 MCP 安全测试 | BLOCKED | DQ-20 |

## 结论

GATE-AI 当前不能通过。已具备可执行的安全基线原型，但正式认证、业务输出 Schema、
外部 MCP、持久恢复、生产 Provider 政策和长期 Trace 仍需负责人批准及实现。
