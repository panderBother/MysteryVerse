# Task 11 批准记录与实施范围

> 历史状态：2026-11-25 初次审计为 BLOCKED，原因是 DQ-18/DQ-19 尚未批准。
> 当前状态：RESOLVED，DQ-18/DQ-19 已按候选推荐范围批准并实施。
> 审批证据：负责人在收到“下一步需要明确批准 DQ-18 与 DQ-19”的结论后，于
> 2026-11-25 明确回复“开始执行吧”。

## 1. 已批准范围

### DQ-18

批准内容包括：

1. `packages/contracts` 是跨服务业务 Schema 唯一权威源。
2. Registry 使用显式 SemVer、Active Version 和追加式版本历史。
3. 不兼容变更必须通过兼容性门禁和新基线评审。
4. Runtime 只能使用已批准 Active Schema 绑定。
5. Replay、Deprecation、Retirement 和保留策略按
   `docs/architecture/contracts-registry.md` 执行。

### DQ-19

批准内容包括：

1. 生产候选为 mTLS + HMAC-SHA256。
2. HMAC canonical 覆盖签名版本、Key ID、timestamp、nonce、Service ID、
   method、path、query 和 body SHA-256。
3. Key ID 支持 Active/Previous 轮换、撤销和 fail-closed。
4. 密钥通过 Key Provider/KMS 抽象获取；当前只实现隔离环境 Provider。
5. 多实例 nonce 使用 Redis 原子 `SET NX EX`；不可用时拒绝请求，不降级放行。
6. 内部授权按 Service Identity + operation/resource/tenant scope，deny by
   default。
7. 安全审计记录身份、Request ID、Key ID、认证/授权结果和 scope，不记录密钥
   或业务正文。

## 2. 未批准范围

以下项目没有被“开始执行”扩展批准：

| 决策 | 当前状态 | 影响 |
| --- | --- | --- |
| DQ-20 | Proposed | 外部 MCP Server、信任等级和开放 egress 继续阻塞 |
| DQ-21 | Proposed | 正式 Run/Checkpoint 持久化、恢复和 owner lease 继续阻塞 |
| DQ-22 | Proposed | 真实 Provider 白名单、区域、预算和 fallback 继续阻塞 |
| ADR-001 | Proposed | Memory 权威写入与正式 Python Memory 协议继续阻塞 |
| ADR-004 | Proposed | `event_seq` / `state_version` 正式语义继续阻塞 |
| ADR-005 | Proposed | 正式暂停/恢复控制协议继续阻塞 |
| ADR-008 | Proposed | 持久 PlayerContext 或正式上下文协议继续阻塞 |

## 3. 实施证据

- Schema Registry：`packages/contracts/registry/schema-registry.json`。
- 兼容性基线：`packages/contracts/compatibility/schema-baseline-v1.json`。
- Registry Runtime：`ai_service/src/mysteryverse_ai/schemas/registry.py`。
- HMAC 与 key 校验：`ai_service/src/mysteryverse_ai/api/request_integrity.py`。
- Key Provider：`ai_service/src/mysteryverse_ai/security/keys.py`。
- Nonce Store：`ai_service/src/mysteryverse_ai/security/nonce.py`。
- ACL：`ai_service/src/mysteryverse_ai/security/acl.py`。
- 审计：`ai_service/src/mysteryverse_ai/security/audit.py`。
- TLS 配置：`ai_service/src/mysteryverse_ai/security/tls.py`。
- Go 客户端：`backend/internal/ai/client/client.go`。
- GATE 状态：`docs/architecture/gate-ai-checklist.md`。
- 验证报告：`docs/architecture/task-11-report.md`。

## 4. 生产未验证项

- 真实 CA 签发、吊销、短周期轮换和跨环境信任链。
- 真实 KMS/Secret Manager 和签名密钥自动轮换。
- 多实例 Redis nonce 集成、故障切换和容量压测。
- 生产 mTLS 端到端部署、证书过期告警和吊销传播。
- 外部 MCP、真实 Provider、Run/Checkpoint 和正式业务 Agent Schema。

这些能力只能标记 `NOT VERIFIED` 或 `BLOCKED`，不能因单元测试或 Mock 测试而
声明生产通过。
