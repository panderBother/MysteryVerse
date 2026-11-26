# Task 14 CI Pipeline 报告

> 日期：2026-11-26
> CI Platform：GitHub Actions
> Workflow：`IMPLEMENTED`
> CI Execution：`NOT VERIFIED`（尚未推送或触发）
> Production Verification：`NOT VERIFIED`
> Task 15 更新：首次执行前审计已完成，当前等待负责人明确授权 commit/push；
> 证据见 `task-15-first-ci-execution.md`。

## 1. Preflight Audit

- Git 远端为 GitHub：`panderBother/MysteryVerse`。
- Task 14 前没有 `.github/workflows`，也没有其他 CI 配置。
- 已有 DQ-21 Unit、MySQL/Redis Integration 和双 Worker 测试入口。
- 已有 Migration Up/Down/Up 测试代码。
- 已有 Backup/Restore Probe 与隔离数据库验证入口。
- 本机仍无 Docker/MySQL/Redis。

## 2. CI Platform

选择 GitHub Actions，原因：

- 仓库已在 GitHub。
- 支持固定版本 MySQL/Redis Service Containers。
- 支持 Ubuntu Runner 上的多进程 Worker 测试。
- 支持 Artifact、JUnit、Job Summary 和最小权限 Token。

没有迁移 Git 平台，也没有引入第二套 CI。

## 3. CI Workflow

新增：

```text
.github/workflows/dq21-infrastructure-verification.yml
```

Jobs：

- Unit and Static Verification。
- MySQL Redis Integration and Multi-Worker。
- Backup and Restore Verification。
- DQ-21 Verification Summary。

## 4. MySQL / Redis Configuration

- MySQL `8.0.36`。
- Redis `7.2.7`。
- MySQL isolated database `mysteryverse_ci`。
- `mv_ai` isolated test schema。
- Health checks、Job timeout、container diagnostics 已配置。
- CI 使用临时测试密码，不读取生产 Secret。

## 5. Integration Tests

CI 执行：

- Migration Up/Down/Up/Down。
- Run/Step/Step Attempt/Recovery Attempt/Checkpoint/Idempotency Persistence。
- Checkpoint AES-GCM、Hash、Schema。
- DB Lease Expiry + Fencing。
- Recovery Attempt、Checkpoint Recovery、Cleanup。

## 6. Multi-Worker Tests

CI 使用两个独立 `python -m ...persistence_probe` 进程：

- Worker A：获取 Token N，等待 Lease 过期。
- Worker B：获取 Token N+1、写入 Step、完成 Run。
- Worker A：使用旧 Token N 写入。

预期由真实 MySQL 拒绝 A 写入。测试代码已实现，CI Execution 尚未发生。

## 7. Fencing Token Tests

当前状态：

- Unit：VERIFIED。
- Real DB：`NOT VERIFIED`，等待 CI。
- Stale Owner Rejection：测试已实现，等待 CI。

## 8. Recovery Tests

真实 MySQL/Redis Integration 测试覆盖：

- failed Run 恢复。
- Recovery Attempt。
- Owner/Fence 更新。
- Checkpoint 解密和恢复。
- Recovery Failure 与 Cleanup 数据一致性。

CI Execution：`NOT VERIFIED`。

## 9. Cleanup / Retention

Integration 测试覆盖：

- Active Run 保护。
- Completed Run Eligibility。
- Recovery 完成后的清理。
- Audit 独立保留。
- Idempotency/Checkpoint 与 Run 清理。

CI Execution：`NOT VERIFIED`。

## 10. Backup / Restore

新增 `backup_restore_probe`：

- 生成合成 Run/Step/Checkpoint/Recovery/Audit/Idempotency。
- 两个隔离 MySQL Service。
- 容器内 mysqldump + gzip。
- 恢复后验证 Checkpoint 解密与记录数量。
- 恢复后 Owner 为空，重新 Claim 获取更高 Fence。
- Backup SQL 不上传 Artifact。

CI Execution：`NOT VERIFIED`。

## 11. CI Security

- Workflow Token：`contents: read`。
- 无生产 Secrets。
- PR 不接收生产凭据。
- Backup SQL 不进入 Artifact。
- 只上传 Hash、脱敏日志、JUnit 和合成元数据。
- 不使用真实 AI Provider。

## 12. Failure Diagnostics

已配置：

- Pytest JUnit XML。
- Migration/Integration/Restore log。
- MySQL/Redis container logs。
- Report Job 明确失败，不吞错。

## 13. CI Execution Status

- Workflow 文件：`IMPLEMENTED`。
- YAML 解析：通过。
- `actionlint v1.7.7`：通过。
- 本地无法执行 Service Container：Docker 不存在。
- GitHub Actions：未推送/未触发，`NOT VERIFIED`。

## 14. Verification Evidence

- 本地 `./scripts/mv.ps1 check`：通过。
- Python：63 passed，3 integration/multi-worker skipped。
- mypy：74 个源文件通过。
- Workflow YAML：解析通过。
- PowerShell 验证脚本：语法通过。
- Docker：未安装，真实 Infrastructure 仍未执行。

## 15. Implemented vs Verified

Implemented：

- GitHub Actions DQ-21 CI Workflow。
- MySQL/Redis services。
- Multi-worker integration test。
- Recovery/Cleanup integration test。
- Backup/Restore probe and job。

Verified：

- Unit、static、contracts、Go/Web/Python build。
- CI YAML schema-level parsing。
- Test code compilation and local skip behavior。

Not Verified：

- Hosted CI execution。
- Real Migration。
- Real MySQL/Redis Integration。
- Multi-Worker failover。
- Real stale-Fence rejection。
- Backup/Restore。
- Production。

## 16. Blocked / Remaining Issues

- Workflow 尚未推送触发；不能生成 GitHub Actions Run ID/URL。
- 本地无 Docker，无法执行等价 Service Container 流程。
- DQ-20 仍 BLOCKED。
- DQ-22 仍 PROPOSED。
- ADR-001/004/005/008 仍 PROPOSED。
- AI DM 仍 BLOCKED。

## 17. DQ-21 Gate Recommendation

- Unit Verification：VERIFIED。
- Integration Verification：NOT VERIFIED。
- Multi-Worker Verification：NOT VERIFIED。
- Fencing Token Real DB：NOT VERIFIED。
- Recovery Integration：NOT VERIFIED。
- Cleanup Integration：NOT VERIFIED。
- Backup/Restore：NOT VERIFIED。
- Production Verification：NOT VERIFIED。

## 18. DQ-22 Readiness

当前不具备讨论 DQ-22 实施的条件。必须先推送并成功执行 GitHub Actions，
由负责人审查 CI Run 证据后，再讨论 DQ-22。
