# Task 15 首次 CI 执行与验证关闭报告

> 日期：2026-11-26
> CI Platform：GitHub Actions
> CI Execution：`VERIFIED`
> DQ-21 Hosted CI Verification：`VERIFIED`
> Production Verification：`NOT VERIFIED`

## 1. Preflight Audit

- Branch：`main`。
- Remote：`origin https://github.com/panderBother/MysteryVerse.git`。
- Push 前 Local HEAD 与 Remote：`79e587ad8adf3f67d41ea1d15d8c8ead54cf4a7b`。
- Workflow：`.github/workflows/dq21-infrastructure-verification.yml`。
- Workflow YAML 和 `actionlint v1.7.7`：通过。
- 本地无 `.env`、Token、Private Key 或其他敏感文件。

## 2. Git Submission

Commit：

```text
de2f8e1b273f94e38b54f89f5dd237a9eeb092c6
test(ci): verify concurrent DQ-21 fencing writes
```

Push：

```text
2e14d06..de2f8e1 main -> main
```

远程 `origin/main` 与本地提交一致。

首轮 CI Commit `2e14d06`（Run `45803587621`）已成功；随后补充真实 MySQL
并发写入回归，并用 Commit `de2f8e1` 触发最终 CI。

## 3. Actual CI Run

| Item | Value |
| --- | --- |
| Repository | `panderBother/MysteryVerse` |
| Branch | `main` |
| Commit SHA | `de2f8e1b273f94e38b54f89f5dd237a9eeb092c6` |
| Workflow | `DQ-21 Infrastructure Verification` |
| Run ID | `45968540075` |
| Run URL | https://github.com/panderBother/MysteryVerse/actions/runs/45968540075 |
| Event | `push` |
| Result | `success` |

## 4. Job Results

| Job | Job ID | Result |
| --- | --- | --- |
| Unit and Static Verification | `140403008553` | success |
| MySQL Redis Integration and Multi-Worker | `140403008082` | success |
| Backup and Restore Verification | `140403008445` | success |
| DQ-21 Verification Summary | `140403450539` | success |

## 5. Unit Job Evidence

实际日志：

- Python Unit：`63 passed, 4 deselected`。
- Contract tests：`5 passed`。
- Ruff：`All checks passed`。
- Format：`105 files already formatted`。
- mypy：`Success: no issues found in 74 source files`。
- Go Test/Vet：成功。

Artifact：

- Name：`dq21-unit-evidence`
- ID：`15183748405`
- URL：https://github.com/panderBother/MysteryVerse/actions/runs/45968540075/artifacts/15183748405

## 6. Integration / Multi-Worker Evidence

Job steps 实际执行：

- Initialize MySQL `8.0.36` / Redis `7.2.7` containers：成功。
- Prepare isolated `mv_ai` schema：成功。
- Migration Up/Down/Up/Down test：成功。
- Apply migrated schema：成功。
- MySQL/Redis Integration and Multi-Worker tests：成功。

Pytest 结果：

```text
4 passed, 1 deselected
```

覆盖：

- Run/Step/Step Attempt/Checkpoint/Idempotency Persistence。
- DB Lease Expiry、Fencing Token、双 Owner 冲突。
- Checkpoint AES-GCM、Hash、Schema。
- Recovery Attempt、Checkpoint Recovery、Cleanup。
- 两个独立 Worker Process。
- Worker B 接管后 Token 2 写入成功。
- Worker A 旧 Token 1 写入被真实 MySQL 拒绝。
- 旧/新 Fence 对同一 Run 的真实 MySQL 并发条件更新：新 Owner 成功，旧 Owner 拒绝。

Artifact：

- Name：`dq21-integration-evidence`
- ID：`15183219255`
- URL：https://github.com/panderBother/MysteryVerse/actions/runs/45968540075/artifacts/15183219255

## 7. Backup / Restore Evidence

Job steps 实际执行：

- 两个隔离 MySQL Service 启动并健康。
- Source Database Migration：成功。
- 合成 Run/Step/Checkpoint/Recovery/Audit/Idempotency 数据：成功。
- 容器内 `mysqldump | gzip`：成功。
- gzip 完整性和 SHA-256：成功。
- Restore 到独立 MySQL：成功。
- Checkpoint 解密和数据一致性验证：成功。

验证输出：

```json
{
  "status": "verified",
  "run_id": "01M8E33PSBZZNN6EJQHQAD72FF",
  "counts": {
    "steps": 1,
    "step_attempts": 1,
    "checkpoints": 1,
    "recovery_attempts": 1,
    "audit_records": 1,
    "idempotency_records": 1
  }
}
```

恢复后 Run Owner 为空，不信任备份中的旧 Lease；重新 Claim 会生成更高 Fencing Token。

Artifact：

- Name：`dq21-backup-restore-evidence`
- ID：`15183299142`
- URL：https://github.com/panderBother/MysteryVerse/actions/runs/45968540075/artifacts/15183299142

## 8. DQ-21 Verification Closure

| Category | Status | Evidence |
| --- | --- | --- |
| Unit Verification | VERIFIED | Unit Job + Artifact |
| Hosted CI Execution | VERIFIED | Run `45968540075` success |
| Integration Verification | VERIFIED | Integration Job 4 passed |
| Multi-Worker Verification | VERIFIED | Independent Worker A/B test passed |
| Fencing Token Real DB | VERIFIED | Old Token + concurrent conditional write rejected by MySQL |
| Recovery Integration | VERIFIED | Recovery Attempt/Checkpoint test passed |
| Cleanup Integration | VERIFIED | Cleanup Integration test passed |
| Backup/Restore | VERIFIED | Two-MySQL dump/restore verification |
| Production Verification | NOT VERIFIED | Hosted CI is not production |

## 9. Failure Analysis

首轮 Run `45803587621` 无需修复并成功。随后发现并发 DB 写入验证不足，
新增一个真实 MySQL 并发条件写入测试；最终 Run `45968540075` 全部 Job 成功。

非阻塞日志：

- Go/uv cache 保存时出现竞争提示，但不影响 Job 结果。
- 没有测试失败、跳过关键验证、吞错或 Artifact 上传失败。

## 10. Security / Evidence Boundary

- Workflow Token 仅 `contents: read`。
- 没有生产 Secrets。
- 没有上传 Backup SQL。
- Artifact 只包含 JUnit、脱敏日志、Hash 和合成元数据。
- Checkpoint 明文和密钥没有进入 Artifact。

## 11. DQ-21 Gate Recommendation

- DQ-21：`IMPLEMENTED / HOSTED CI VERIFIED`。
- Production：`NOT VERIFIED`。
- DQ-22：仍为 `PROPOSED`，CI 成功不会自动批准。
- DQ-20：仍 `BLOCKED`。
- ADR-001/004/005/008：仍 `Proposed`。
- AI DM：仍 `BLOCKED`。
- Exactly-once：未宣称。

## 12. Next Recommended Action

可以基于本次 CI 证据讨论 DQ-22 的范围和批准条件，但不得直接实施。进入 DQ-22 前仍需负责人明确批准供应商政策、数据边界、预算和降级合同。
