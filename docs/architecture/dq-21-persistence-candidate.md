# DQ-21 Run / Step / Checkpoint Persistence Candidate

> 状态：`APPROVED / IMPLEMENTED`
> 验证状态：Unit Tests `VERIFIED`；MySQL+Redis Integration、KMS 与 Restore
> `NOT VERIFIED`
> 日期：2026-11-25
> 依据：DQ-21 候选设计、`DATABASE_DESIGN.md` §6.5、Task 12 及 Supplement

## 1. 审计结论

- DQ-18/DQ-19 已批准并完成 Task 11 测试级验证。
- 负责人已批准 MySQL 权威元数据、Redis Owner Lease、DB Fencing、MySQL 加密
  Checkpoint Blob、Retention 与同 Run Recovery 方案。
- 当前只有 `InMemoryAgentRunStore` 与 `CheckpointReference` 原型。
- 没有 `mv_ai` 迁移、正式 Owner Lease、Fencing Token、恢复端点或 LangGraph
  Checkpointer。
- Task 12 Supplement 已补齐 Retention、Cleanup、Recovery 和最终验收候选要求。
  这些参数仍只能作为 Proposed 默认值，不能据此冻结生产 Schema。

正式迁移已新增到 `backend/migrations/00009_ai_runtime_persistence.sql`。
内部 Runtime Schema 不进入公共 Contracts Registry。

## 2. 已完成的候选验证

候选代码位于：

```text
ai_service/src/mysteryverse_ai/agent_runtime/persistence_foundation/
ai_service/src/mysteryverse_ai/persistence/
```

### 2.1 Run 状态机

候选状态：

```text
created -> queued -> running
running -> suspended | recovering | completed | failed | cancelled
suspended -> recovering | failed | cancelled
recovering -> running | failed | cancelled
failed -> recovering (controlled recovery only)
completed/cancelled -> terminal (same state only)
```

终态不可逆；非法跳转和从终态恢复都会拒绝。

### 2.2 Owner Lease / Fencing

- Owner Identity 使用稳定实例 ID，不使用 IP。
- 内存 Store 原型使用原子锁验证并发抢占。
- Lease 过期后允许新 Owner 获取。
- Fencing Token 每个 Run 单调递增。
- 旧 Owner 的过期 Lease 不能续租或写入性校验通过。
- 该实现只用于算法验证；生产必须使用 Redis/MySQL 组合，并在权威 DB 写入层
  再次验证 `owner_id + fencing_token`。

### 2.3 Checkpoint Envelope

候选 Envelope 只定义恢复元数据，不冻结 LangGraph State：

- checkpoint ID / Run ID。
- checkpoint sequence。
- schema version / runtime version。
- status / storage reference。
- payload SHA-256 与可选 key ID。
- created time。

恢复前必须校验状态、Schema 版本和 Payload Hash。未知版本、非 valid 状态和
hash 不匹配均 fail-closed。Checkpoint 正文不进入普通日志或公共 Contracts。

### 2.4 Run Idempotency

- Key scope 使用 `scope + idempotency_key`。
- 相同请求 hash 返回原 Run ID。
- 相同 key、不同请求 hash 返回冲突。
- 当前仍是内存候选；正式唯一约束继续采用 Database Design 中的
  `task_type + idempotency_scope + idempotency_key`。

### 2.5 Retry Policy

- 只重试显式分类的可重试错误。
- 检查最大尝试次数与 deadline。
- 指数退避、最大延迟和有界 jitter。
- 有外部副作用的工具不得仅依据 Retry Policy 自动重试。

### 2.6 Retention / Cleanup Eligibility

候选 RetentionPolicy 可配置，默认值：

| Resource | Default |
| --- | --- |
| Active Run | 不自动过期 |
| Completed Run | 30 天 |
| Failed Run | 90 天 |
| Cancelled Run | 30 天 |
| Completed Checkpoint | 30 天 |
| Failed Checkpoint | 90 天 |
| Cancelled Checkpoint | 30 天 |
| Runtime Audit | 180 天 |
| Idempotency Record | 7 天 |

Cleanup Eligibility 会检查终态时间、活跃引用和 Recovery 状态。Active Run 永远
不可清理；Retention 到期本身不等于立即删除。统一 Cleanup Worker、Batch Delete
和独立 Cleanup Job 元数据尚未实现。

### 2.7 Recovery Attempt

候选 `RecoveryAttempt` 记录：

- Recovery Attempt ID、Run ID、Attempt Number。
- Trigger、Recovery Checkpoint。
- Previous Owner、New Owner、Fencing Token。
- Status、Started/Finished、Failure Category。

恢复准入要求 Run 为 failed/suspended/recovering、Checkpoint 可用且有效、Schema
兼容，并且能获取 Owner Lease。Recovery Attempt 不覆盖原失败记录。

### 2.8 Checkpoint Size

候选上限为 Payload 1 MiB，策略可配置。超限返回稳定错误，不覆盖旧 Checkpoint，
也不改变 Run 状态。正式实现仍需验证压缩后大小和数据库 Blob 限制。

## 3. 推荐正式方案

以下内容仍需负责人批准，当前未冻结：

1. MySQL `mv_ai.agent_runs` 保存 Run 元数据，`agent_steps` 保存 Step 记录，
   `agent_checkpoints` 保存 Checkpoint 元数据和加密引用。
2. Redis 保存短期 Owner Lease；MySQL Run 行保存当前 Fencing Token。
3. 所有 Run/Step/Checkpoint 写入都在 MySQL 事务内比较 owner + fencing token。
4. Checkpoint Payload 首版使用 `MEDIUMBLOB` 加密保存，默认上限 1 MiB；超过阈值
   时拒绝或采用单独批准的拆分方案，不引入对象存储依赖。
5. Run 状态迁移由统一状态机执行，数据库使用条件更新防止并发覆盖。
6. 故障恢复沿用同一 Run ID 并创建 Recovery Attempt；用户主动重新发起任务创建
   新 Run，可引用 Parent/Recovery Source，但不覆盖历史 Run。
7. 外部副作用采用 at-least-once 执行 + 幂等 Completion Record，不承诺无条件
   exactly-once。
8. Active/Completed/Failed/Cancelled Run 与 Checkpoint 按 RetentionPolicy 清理；
   Audit 与 Idempotency 独立保留。
9. Cleanup 使用可配置 Batch Size，失败可安全重试，不得通过 Cascade Delete
   绕过 Recovery、Audit 和引用检查。

## 4. 生产故障策略候选

- MySQL 不可用：停止新的权威状态写入，Run 不得伪成功。
- Redis 不可用：停止新的 Lease 获取与续租；已有 Owner 不得在无法确认 Lease
  的情况下继续权威写入。
- Lease 丢失/过期：旧 Owner 后续写入必须被 Fencing Token 拒绝。
- Checkpoint Hash/版本错误：拒绝恢复，返回 `CHECKPOINT_UNAVAILABLE`；不得静默
  从错误状态继续。
- KMS 不可用：拒绝加密写入/解密恢复，不回退到明文。
- 进程崩溃：新 Owner 获取更高 Fencing Token，再从最近 valid Checkpoint 恢复
  未完成 Step。

## 5. 最终验收状态

| 验收域 | 当前状态 | 证据 |
| --- | --- | --- |
| Run/Step/Checkpoint Persistence | PASS（Unit） | Repository、Runtime Store 与 `mv_ai` 迁移已实现；真实 MySQL 未运行 |
| MySQL Migration | IMPLEMENTED / NOT VERIFIED | Goose migration 与 schema assertions 已加入；无 Docker/MySQL 环境执行 |
| Schema Version / Checkpoint Integrity | VERIFIED（Candidate） | Envelope、version、hash 测试 |
| Owner Lease / Expiry | VERIFIED（Candidate） | 并发抢占、过期和续租测试 |
| Fencing Token | VERIFIED（Candidate） | 单调 token 与旧 Owner 拒绝 |
| Recovery Attempt | VERIFIED（Candidate） | 模型、准入和 incompatible 拒绝测试 |
| Run/Step Recovery | PASS（Candidate/Unit） | 同 Run Recovery Coordinator、Recovery Attempt 与 Lease Conflict 测试 |
| Multi-instance Recovery | BLOCKED | 未运行双 Worker/Redis/MySQL 集成测试 |
| Retention Policy | VERIFIED（Candidate） | Active/terminal/checkpoint/idempotency 判定测试 |
| Cleanup Worker | PASS（Unit） | 周期 Worker、Batch、Job、Audit 与失败处理已实现；真实 DB 未运行 |
| Encryption / KMS | PASS（Unit）/ NOT VERIFIED | AES-GCM、Key ID、压缩和完整性已测试；真实 KMS 未部署 |
| Backup / Restore | NOT_VERIFIED | 真实恢复测试未执行 |
| Observability | NOT_IMPLEMENTED | 尚无 Run/Lease/Fencing/Cleanup 生产指标 |

## 6. 验证证据

```powershell
uv run --project ai_service pytest ai_service/tests/test_persistence_foundation.py
uv run --project ai_service pytest ai_service/tests/test_runtime_persistence_formal.py
uv run --project ai_service pytest ai_service/tests
uv run --project ai_service ruff check ai_service
uv run --project ai_service mypy ai_service/src
```

当前结果：

- Persistence foundation + formal persistence 测试：13 项通过。
- Python 全量测试：63 项通过，真实 MySQL/Redis Integration/双 Worker 3 项条件跳过。
- Ruff：通过。
- mypy：74 个源文件通过。

## 7. 批准记录

负责人于 2026-11-25 批准：

1. MySQL 权威元数据 + Redis Lease + DB Fencing。
2. Checkpoint 加密正文首版使用 MySQL Blob，预留对象存储迁移接口。
3. 默认 Retention 策略与可配置要求。
4. 同 Run ID Recovery + Recovery Attempt + Step Attempt。
5. Cleanup Eligibility、Batch、Retry、Audit 与 Monitoring 要求。
6. DQ-20/DQ-22 不在本阶段实施，AI DM 继续 BLOCKED。

尚未验证：

- 真实 MySQL/Redis 双实例接管。
- 真实 KMS 与证书化密钥轮换。
- Backup/Restore 和 RTO/RPO。
- LangGraph Graph Thread 自动恢复执行。
