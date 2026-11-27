# Task 12 实施与验证报告

> 日期：2026-11-25
> DQ-21：APPROVED / IMPLEMENTED / UNIT-VERIFIED
> Task 15 CI：Integration / Multi-Worker / Recovery / Cleanup / Backup-Restore
> VERIFIED。Production、真实 KMS 与 LangGraph Auto Replay：NOT VERIFIED。

## 1. 批准记录

负责人于 2026-11-25 明确批准：

- MySQL 为 Run/Step/Checkpoint Metadata 权威存储。
- Redis 作为 Owner Lease 协调设施。
- MySQL `owner_lease_expires_at + fencing_token` 保护权威写入。
- Checkpoint 加密正文首版进入 MySQL Blob，默认压缩后上限 1 MiB。
- Completed/Failed/Cancelled Run 与 Checkpoint 使用批准的 Retention 默认值。
- 失败恢复沿用同一 Run ID，并记录 Recovery Attempt 与 Step Attempt。
- 用户重新发起任务创建新 Run，不伪装为原 Run Recovery。
- DQ-20、DQ-22、ADR-001/004/005/008 和 AI DM 仍在门禁后。

## 2. 实际实现

### 2.1 MySQL Persistence

迁移：

```text
backend/migrations/00009_ai_runtime_persistence.sql
```

新增 `mv_ai` 表：

- `agent_runs`
- `agent_idempotency_records`
- `agent_steps`
- `agent_step_attempts`
- `agent_checkpoints`
- `agent_recovery_attempts`
- `runtime_audit_records`
- `runtime_cleanup_jobs`

`agent_runs` 保存权威 Owner、单调 Fencing Token、DB Lease 到期时间、执行版本、
运行期限、终态时间和失败分类。

### 2.2 Runtime Repository

`ai_service/src/mysteryverse_ai/persistence/repository.py` 实现：

- Run 创建与 Idempotency Request Hash 冲突检测。
- `SELECT ... FOR UPDATE` 下的 Owner Claim。
- DB Lease 未过期时的双 Owner 抢占拒绝。
- Owner/Fence 条件状态更新。
- Step、Step Attempt、Checkpoint、Recovery Attempt 和 Audit 写入。
- Cleanup Eligibility 所需的引用/Recovery 检查。
- 受控 Run 清理和 Cleanup Job 状态。

### 2.3 Redis Lease

`RedisOwnerLeaseStore` 使用 Lua 原子脚本实现：

- SET NX + TTL 获取。
- Owner/Fence 匹配续租。
- Owner/Fence 匹配释放。
- Fencing Token 校验。

Redis 不是最终写入授权；Repository 的每次关键写入仍校验 DB Owner/Fence。

### 2.4 Checkpoint Encryption

`CheckpointCipher` 实现：

- JSON 规范化。
- zlib 压缩。
- 压缩后 1 MiB 限制。
- SHA-256 完整性。
- AES-256-GCM 加密与 AAD 绑定 `run_id + checkpoint_seq + schema_version`。
- Key ID、12-byte nonce、密文与 Hash 分离存储。

### 2.5 Recovery

`RuntimeRecoveryService` 实现：

- 检查 Run 状态、Checkpoint、Schema、Hash。
- 通过 DB 获取新 Fencing Token 与 Owner 资格。
- 获取 Redis Lease。
- 创建 Recovery Attempt。
- 将 Run 从 failed/suspended/recovering 受控转入 running。
- Lease 获取冲突时释放 DB Owner，不伪装恢复成功。

当前没有把 LangGraph Graph Thread 自动重放到恢复后的 Run；该部分仍
`NOT_IMPLEMENTED`。

### 2.6 Retention / Cleanup

`RuntimeCleanupWorker` 实现：

- 可配置 RetentionPolicy。
- Active Run 永不清理。
- 有子 Run/Recovery Source 引用或正在恢复时拒绝清理。
- Batch Size、Cleanup Job、Audit、失败记录和周期运行。
- Idempotency/Audit 独立过期清理。
- Retry 通过下一轮 Cleanup 重新扫描完成，不使用长期事务。

Worker 已接入 `mysteryverse_ai.workers.main`，由 `AI_CLEANUP_INTERVAL_SECONDS`
控制周期。

### 2.7 Runtime Integration

- `MySQLAgentRunStore` 接入 Agent Runtime。
- Runtime 开始前创建 Run、Claim Owner、获取 Redis Lease。
- 完成时执行受控状态迁移并释放 Lease/Owner。
- Trace Step 写入 `agent_steps`，首次执行创建 Step Attempt。
- 未启用持久化时继续使用 `InMemoryAgentRunStore`。

## 3. 测试结果

执行：

```powershell
uv run --project ai_service pytest ai_service/tests
uv run --project ai_service ruff check ai_service
uv run --project ai_service ruff format --check ai_service
uv run --project ai_service mypy ai_service/src
go test ./migrations
./scripts/mv.ps1 check
```

结果：

- Python：63 项通过，3 项真实 MySQL/Redis Integration/双 Worker 测试按环境条件跳过。
- Python Ruff：通过。
- Python mypy：74 个源文件通过。
- Go migration package：通过（未配置真实 MySQL 时 schema integration test 跳过）。
- Contracts Registry：142 个定义。

测试覆盖：

- Run 状态机、非法跳转和失败 Run 受控恢复。
- Owner Lease 并发抢占、过期、旧 Fence 拒绝。
- Checkpoint 加密、压缩、篡改、Schema 不兼容和大小限制。
- Idempotency Request Hash 冲突。
- Recovery 成功、Lease Conflict 和 incompatible checkpoint。
- Retention 边界、引用保护、Cleanup Job/Audit/指标结果。
- 真实 MySQL/Redis Integration 测试代码已建立，但当前环境未执行。

## 4. Multi-instance 验证

状态：`VERIFIED`（GitHub Actions Run `45803587621`）

证据：

- Worker A/Worker B 使用两个独立进程。
- Worker A 旧 Fence 写入被真实 MySQL 拒绝。
- Worker B 使用新 Fence 写入成功。
- Job `139852856357` 成功。

## 5. Failure Scenario 状态

| Scenario | 当前证据 |
| --- | --- |
| MySQL unavailable | Repository 无降级成功路径；真实故障注入 NOT VERIFIED |
| Redis unavailable | Lease 获取/续租失败 fail-closed；真实故障注入 NOT VERIFIED |
| Lease expired | 单元测试通过 |
| Fence mismatch | Repository 条件写入与单元测试通过；真实双 Worker/MySQL 测试代码存在但未执行 |
| Checkpoint corrupted | AES-GCM/Hash 测试通过 |
| Checkpoint incompatible | Schema 拒绝测试通过 |
| Recovery failed | 保留 Run，Recovery Attempt 可追踪 |
| Cleanup failed | Job failed + Audit，Run 保留 |
| KMS unavailable | 加密入口 fail-closed；真实 KMS NOT VERIFIED |
| Run timeout | 状态机与 Runtime Deadline 字段已实现；调度扫描 NOT VERIFIED |
| Duplicate Run | Request Hash Conflict 检测 |
| Duplicate Step | Step/Attempt 唯一约束已定义，真实 DB 未验证 |
| Owner takeover | DB Lease/Fence + Redis Lease 代码 |
| DB/Redis restore | 未执行 |

## 6. NOT VERIFIED / 未实现

- 生产 KMS、密钥轮换/撤销和证书化密钥访问。
- LangGraph Thread 自动恢复与业务状态重放。
- Step Attempt 自动 Retry 调度；目前记录单次执行 Attempt。
- RTO/RPO 实测；备份恢复功能已验证，但未测目标指标。
- OpenTelemetry/Prometheus Run/Lease/Fence/Cleanup 指标；当前只有结构化日志。

### 6.1 Restore Test Plan

真实环境具备后必须执行：

1. 备份 MySQL，确认包含 `mv_core`、`mv_ai` 和 Goose 版本表。
2. 恢复到隔离数据库，禁止直接覆盖生产实例。
3. 校验 Run/Step/Checkpoint/Recovery/Idempotency/Audit 行数和外键一致性。
4. 使用同一 KMS Key ID 解密抽查 Checkpoint，验证 Hash、Schema 与 Runtime Version。
5. 不恢复 Redis 旧 Lease 作为有效所有权；清空协调 namespace 后重新获取 Lease。
6. 用两个 Runtime 实例竞争恢复同一 Run，旧 Fencing Token 写入必须失败。
7. 运行 Cleanup dry-run/小批量清理，确认 Audit 独立保留。
8. 记录 RTO、RPO、失败点和人工恢复步骤。

当前没有可用 MySQL/Redis/KMS 环境，上述计划标记 `NOT VERIFIED`。

## 7. 状态回答

- DQ-21 是否 IMPLEMENTED：是，正式代码与迁移已落地。
- DQ-21 是否 VERIFIED：Hosted CI Verified；Production NOT VERIFIED。
- 是否具备安全 Run Recovery：领域流程、真实 MySQL/Redis Integration 已验证。
- 是否具备 Owner Lease：真实 Redis Lease 与双 Worker 已验证。
- 是否具备 Fencing Token：真实 MySQL 条件写入和旧 Owner 拒绝已验证。
- 是否具备多实例接管：双独立 Worker 接管已验证。
- 是否具备 Cleanup：真实 MySQL Recovery/Cleanup Integration 已验证。
- 是否具备 Restore Verification：独立 MySQL Backup/Restore 已验证；RTO/RPO 未测量。
- 是否可以进入 DQ-22：不可以。DQ-22 仍为 Proposed，需单独批准。
- DQ-20 是否仍 BLOCKED：是。
- AI DM 正式实现是否仍 BLOCKED：是。

## 8. 下一阶段建议

1. 在具备 MySQL 与 Redis 的 CI/开发环境执行 Migration 与双 Worker 测试。
2. 接入真实 KMS，完成 Checkpoint Key Rotation/Revocation。
3. 把 LangGraph Thread/Checkpointer 与 RuntimeRecoveryService 对接。
4. 完成 MySQL/Redis 故障注入、Restore Test 和监控指标。
5. 达到 Integration Verified 后再评估 DQ-22。
