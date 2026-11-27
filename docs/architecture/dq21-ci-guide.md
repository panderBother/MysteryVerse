# DQ-21 CI Verification Guide

> 平台：GitHub Actions
> Workflow：`.github/workflows/dq21-infrastructure-verification.yml`
> Infrastructure Verification 只有 GitHub Actions 实际成功后才会标记 `VERIFIED`。

## 1. 触发方式

Workflow 在以下情况触发：

- Push 到 `main`，且涉及 AI Runtime、Migration、MySQL Init 或 DQ-21 脚本。
- Pull Request 到 `main`，涉及相同路径。
- 手动 `workflow_dispatch`。

权限仅包含：

```yaml
permissions:
  contents: read
```

## 2. Jobs

| Job | 内容 |
| --- | --- |
| `unit` | Python/Contracts/Go 单元测试、Ruff、mypy、Go Vet |
| `integration` | MySQL 8.0.36 + Redis 7.2.7、Migration、Persistence、Recovery/Cleanup、多 Worker |
| `backup-restore` | 两个隔离 MySQL 实例、实际 dump/restore、Checkpoint 解密和数据一致性 |
| `report` | 汇总各 Job 结果；任一依赖失败则 Report Job 失败 |

## 3. MySQL / Redis

- MySQL：`mysql:8.0.36`。
- Redis：`redis:7.2.7`。
- 数据库：`mysteryverse_ci`。
- AI schema：`mv_ai`。
- CI 密码是隔离的临时测试值，不是生产 Secret。
- 不向 PR Workflow 注入生产凭据。

## 4. Integration Job

执行顺序：

1. 创建 `mv_ai` 测试 Schema。
2. Migration Up -> Down -> Up -> Down 测试。
3. 再执行 Migration Up 供 Integration 使用。
4. 运行真实 MySQL/Redis Persistence 测试。
5. 启动两个独立 Python Worker Process。
6. Worker A 获取旧 Fence，Lease 过期后 Worker B 接管。
7. Worker B 写入成功；Worker A 旧 Fence 写入必须被 MySQL 拒绝。
8. 运行 Recovery Attempt、Checkpoint Recovery 与 Cleanup Integration。

日志与 JUnit 上传到 `dq21-integration-evidence` Artifact。

## 5. Backup / Restore Job

使用两个 MySQL Service：

- Source：`127.0.0.1:3306`
- Restore：`127.0.0.1:3307`

流程：

1. Source 应用 Migration。
2. 写入合成 Run、Step、Checkpoint、Recovery、Audit、Idempotency 数据。
3. 通过容器内 `mysqldump | gzip` 备份。
4. 校验 gzip 完整性并生成 SHA-256。
5. 将备份复制到隔离 Restore MySQL。
6. 验证数据数量、Checkpoint AES-GCM 解密、状态与 Fence。
7. 确认恢复后的 Owner 为空，不信任备份中的旧 Lease。
8. 新 Owner Claim 必须获得更高 Fencing Token。

Backup SQL 不进入 Artifact；只上传元数据、Hash、验证日志与容器诊断日志。

## 6. Failure Diagnostics

Workflow 失败时上传：

- Migration log。
- Pytest JUnit XML。
- Integration log。
- MySQL/Redis container logs。
- Backup/Restore verification log。
- 不含密码、Checkpoint 明文或 Backup SQL。

Report Job 在任一依赖非 `success` 时失败，不吞错。

## 7. 状态规则

- Workflow 文件存在：`IMPLEMENTED`。
- `actionlint v1.7.7` 静态语义检查：通过。
- GitHub Actions Run `45803587621`：success。
- Integration / Multi-Worker / Recovery / Cleanup / Backup-Restore：
  Hosted CI `VERIFIED`。
- Hosted CI 成功不等于 Production Verification。
- DQ-22 仍为 Proposed，CI 成功不会自动批准 DQ-22。

## 8. 本地与 CI 关系

本地 Windows 没有 Docker，因此：

- `dq21-infra.ps1` 和 `dq21-verify.ps1` 用于具备 Docker 的本机/CI Runner。
- GitHub Actions 使用 Service Containers，不依赖开发者本机 Docker。
- Local Docker Verification 与 Hosted CI Verification 分别记录。
