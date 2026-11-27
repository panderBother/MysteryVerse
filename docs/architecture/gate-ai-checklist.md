# GATE-AI 准入检查清单

> 日期：2026-11-25
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
| mTLS 基础实现 | PASS（Test） | Go 双向证书握手、缺失/过期客户端证书测试；真实 CA 部署 NOT VERIFIED |
| Key ID / 签名版本 | PASS | `mv-hmac-sha256-v1`、Active/Previous、未知/撤销 Key 拒绝测试 |
| KMS/Key Provider 抽象 | PASS（Test） | Environment Provider 接口与故障边界；真实 KMS NOT VERIFIED |
| 时间戳与 nonce 重放防护 | PASS | 内存与 Redis Store 原子消费、跨 App 重放、Store 故障 fail-closed 测试 |
| 双密钥轮换窗口 | PASS | primary/previous 轮换和撤销测试 |
| 正式服务 ACL scope | PASS | operation/resource/tenant scope、拒绝和跨租户测试 |
| 安全审计 | PASS | 身份、Request ID、Key ID、认证/授权结果和 scope；无密钥/正文 |
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
| 正式 Run 持久化 | PASS | 最终 CI Run `45968540075` 真实 MySQL Migration/Integration 通过 |
| Checkpoint 加密与恢复 | PASS | CI 真实 AES-GCM Checkpoint、Hash/Schema 与 Recovery Integration 通过 |
| LangGraph Checkpoint 自动恢复 | NOT_IMPLEMENTED | Checkpoint Envelope/Persistence 已实现；Graph Thread 自动重放尚未接入 |
| 多实例并发隔离 | PASS | 双独立 Worker Process、Owner 冲突、新 Fence 接管和旧 Fence DB 拒绝通过 |
| 并发 DB Fence 写保护 | PASS | 真实 MySQL 并发条件更新：旧 Owner 拒绝、新 Owner 成功 |
| Recovery Attempt / 准入 | PASS | CI 真实 MySQL/Redis Recovery Attempt、Checkpoint Recovery 通过 |
| Retention Eligibility | PASS | Unit + CI 真实 Cleanup Integration 通过 |
| Cleanup Worker | PASS | CI 真实 MySQL Batch/Recovery Cleanup、Audit 保留通过 |
| Backup / Restore | PASS | 双 MySQL dump/restore、Checkpoint 解密和记录一致性验证通过 |
| 故障重试与降级 | PASS | Provider 有限重试、Tool 失败、Timeout 测试 |
| Trace 关联 | PASS | request/trace/run/step 关联 |
| Trace 长期持久化 | NOT_IMPLEMENTED | 未写 `mv_ai`，未接 OpenTelemetry |

## 输出约束

| 检查项 | 状态 | 证据/阻断原因 |
| --- | --- | --- |
| 版本化任务级 Schema Registry | PASS | 142 定义 Registry、兼容基线和 Active 绑定测试 |
| 模型不能选择 Schema | PASS | Go 请求指定，模型只生成 output |
| 必填/枚举/范围/嵌套校验 | PASS | 通用 Pydantic Schema 原型 |
| 未知字段拒绝 | PASS | `extra=forbid` 测试 |
| 校验失败不得成功返回 | PASS | Runtime 返回 failed + validation_errors |
| 非法输出错误可追踪 | PASS | 响应 validation_errors 和 trace |
| AI DM/NPC/生成正式业务 Schema | BLOCKED | DQ-18 治理已批准，但具体业务 Schema 仍需逐项批准 |

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
| Python 单元测试 | PASS | CI Unit 63 passed；Contracts 5 passed |
| MySQL/Redis Integration | PASS | CI 4 passed：Persistence、Recovery/Cleanup、Multi-Worker、并发 Fence |
| FastAPI 集成测试 | PASS | health、agent、认证、签名、幂等 |
| Go/Python 契约测试 | PASS | 142 definitions、三端 examples |
| 真实跨进程协议测试 | PASS | Go Client → FastAPI Mock Provider，Token 与 HMAC 模式均通过 |
| 权限测试 | PASS | Tool permissions/actor/trust/confirmation |
| 认证篡改/重放测试 | PASS | signed HTTP tests |
| 超时/取消/失败测试 | PASS | Runtime tests |
| 持久化/Checkpoint 故障注入 | PASS | 旧 Fence、Lease Expiry、Checkpoint Tamper/Schema、Recovery/Cleanup 故障路径已测试 |
| mTLS/多实例 nonce 测试 | PASS（Test） | Go mTLS 握手、共享 Store 协议与 Redis Store 原子消费；真实部署 NOT VERIFIED |
| SSRF/真实 MCP 安全测试 | BLOCKED | DQ-20 |

## 结论

GATE-AI 当前不能通过。DQ-18/DQ-19 已提供测试验证的 Registry 与安全基础；
DQ-21 Hosted CI 已通过真实 MySQL/Redis、Multi-Worker、Recovery/Cleanup 和
Backup/Restore。业务输出 Schema、外部 MCP、Production KMS/发布、真实 Provider
政策、Graph Auto Replay 和长期 Trace 仍需要批准、部署或实现。
