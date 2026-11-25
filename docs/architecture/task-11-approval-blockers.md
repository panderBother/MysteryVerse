# Task 11 批准状态审计与实施阻断

> 日期：2026-11-25
> 审计结论：BLOCKED
> 原因：`DQ-18` 与 `DQ-19` 未获负责人批准

## 1. 实际批准状态

| 决策 | 仓库状态 | 是否可实施正式协议 |
| --- | --- | --- |
| DQ-18 正式 Schema 与 Contracts 治理 | PROPOSED，Registry 原型 IMPLEMENTED/VERIFIED | 否 |
| DQ-19 mTLS + HMAC 服务安全 | PROPOSED，HMAC 原型 IMPLEMENTED/VERIFIED，mTLS BLOCKED | 否 |
| DQ-20 MCP 信任 | PROPOSED，外部 MCP BLOCKED | 否 |
| DQ-21 Run/Checkpoint 持久化 | PROPOSED，持久化 BLOCKED | 否 |
| DQ-22 真实 Provider 政策 | PROPOSED，真实供应商 BLOCKED | 否 |
| ADR-001 | Proposed | 否 |
| ADR-004 | Proposed | 否 |
| ADR-005 | Proposed | 否 |
| ADR-008 | Proposed | 否 |

仓库没有 `DQ-18 APPROVED` 或 `DQ-19 APPROVED` 记录。Task 11 标题中的
“批准后实施”不能替代负责人对决策内容的明确批准。

## 2. 禁止实施内容

- 不冻结正式 Schema Inventory、Versioning、Active、Compatibility、Replay 或 Lifecycle 协议。
- 不将业务输出 Schema 批量迁入正式 Contracts。
- 不修改生产 mTLS、证书、HMAC canonical、Key ID、KMS、nonce 或 ACL 正式协议。
- 不新增数据库迁移或正式 Run/Checkpoint 结构。
- 不开放外部 MCP。
- 不接真实 Provider。
- 不解除 ADR-001/004/005/008 对应门禁。

## 3. 已存在的独立安全验证

以下 Task 9 原型可以继续作为证据，但不等于 DQ `APPROVED`：

- 可选 HMAC/timestamp/nonce、防重放和双密钥轮换。
- 版本化 Demo Schema Registry、严格 Pydantic 校验。
- D3/D4 Context 默认阻断。
- 本地 Tool 权限、trust、确认和结果上限。
- 进程内 Run/CheckpointReference 元数据。
- Go Client → Python FastAPI Token/HMAC 跨进程测试。

## 4. 批准所需最小输入

### DQ-18

1. 正式 Contracts 是否作为全部业务 Schema 唯一源。
2. Schema Version 格式、保留期限、默认 active 和旧版本重放规则。
3. 业务 Schema 的审批流程与不兼容变更流程。

### DQ-19

1. 是否生产强制 mTLS + HMAC。
2. 证书、KMS、Key ID、nonce 共享存储和轮换策略。
3. Service ACL scope、跨租户规则和审计字段。
4. 开发环境例外边界。

## 5. 解除阻塞后的实施顺序

1. 更新 DQ-18/DQ-19 状态为 `APPROVED` 并记录证据。
2. 实现正式 Contracts Registry、版本和兼容检查。
3. 实现 mTLS/KMS/Key ID/共享 nonce/ACL。
4. 完成失败、多实例和集成测试。
5. 更新 GATE-AI 证据。
6. DQ-18/DQ-19 均达到 `IMPLEMENTED + VERIFIED` 后，才讨论 DQ-21。

## 6. Current Conclusion

- DQ-18：`PROPOSED`，未 `APPROVED`，正式实施 `BLOCKED`。
- DQ-19：`PROPOSED`，未 `APPROVED`，正式实施 `BLOCKED`。
- GATE-AI：未通过。
- AI DM：禁止正式实现。
- 仓库可以发布已测试的审计与安全基线，但不得标记 Task 11 功能完成。
