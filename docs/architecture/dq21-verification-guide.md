# DQ-21 Infrastructure Verification Guide

> 适用范围：本地开发与 CI 验证，不代表 Production Verification。
> 依赖：Docker Desktop/Engine、PowerShell、MySQL 8.0.36、Redis 7.2.7。

## 1. 安全边界

- 只使用隔离测试数据库 `mysteryverse_dq21_test`。
- `dq21-infra.ps1 reset-test-db` 会删除并重建该测试库。
- `dq21-infra.ps1 destroy` 会删除 `mysql_data`、`redis_data` 卷，必须显式传入
  `-ConfirmDeleteVolumes`。
- 不要对生产 DSN 设置 `MYSQL_MIGRATION_TEST_DSN`。
- Redis 旧 Lease 不得直接恢复为有效所有权。

## 2. 启动与检查

```powershell
Copy-Item .env.example .env
./scripts/dq21-infra.ps1 check
./scripts/dq21-infra.ps1 up
```

停止但不删除卷：

```powershell
./scripts/dq21-infra.ps1 down
```

删除本地基础设施卷：

```powershell
./scripts/dq21-infra.ps1 destroy -ConfirmDeleteVolumes
```

## 3. 隔离测试库

```powershell
./scripts/dq21-infra.ps1 reset-test-db -ConfirmTestDatabaseReset
```

该命令重建：

- `mysteryverse_dq21_test`
- `mv_ai`
- 应用账号对两个 Schema 的开发权限

## 4. Migration 验证

```powershell
$env:MYSQL_MIGRATION_TEST_DSN = "root:<root-password>@tcp(127.0.0.1:3306)/mysteryverse_dq21_test?parseTime=true"
Push-Location backend
go test ./migrations -run TestMigrationsUpStatusAndDown -count=1 -v
Pop-Location
```

验证内容：

- Fresh Up。
- Repeated Up。
- Index/Unique/FK/Schema 数量。
- Down 到零。
- Down 后重新 Up。
- `mv_ai` 8 张表存在。

## 5. Real MySQL / Redis Integration

```powershell
$env:AI_RUNTIME_TEST_DATABASE_URL = "mysql+asyncmy://root:<root-password>@127.0.0.1:3306/mysteryverse_dq21_test"
$env:AI_RUNTIME_TEST_REDIS_URL = "redis://:<redis-password>@127.0.0.1:6379/0"
uv run --project ai_service pytest `
  ai_service/tests/test_runtime_persistence_mysql.py `
  ai_service/tests/test_runtime_persistence_multiworker.py `
  -m integration -v
```

验证：

- Run/Step/Checkpoint/Recovery/Idempotency 持久化。
- Checkpoint AES-GCM、Hash、Schema。
- DB Lease Expiry + Fencing。
- Worker A/Worker B 双进程接管。
- Stale Owner 旧 Fence 写入被 MySQL 拒绝。

一键执行：

```powershell
./scripts/dq21-verify.ps1 -ResetTestDatabase
```

## 6. Backup

在 MySQL 容器内生成压缩备份，避免 PowerShell 管道破坏二进制：

```powershell
docker compose exec -T mysql sh -c `
  'mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction --routines --triggers --databases mysteryverse_dq21_test mv_ai | gzip > /tmp/dq21-backup.sql.gz'
docker compose cp mysql:/tmp/dq21-backup.sql.gz .local/dq21-backup.sql.gz
Get-FileHash .local/dq21-backup.sql.gz -Algorithm SHA256
```

## 7. Restore

必须使用独立 MySQL 实例，例如：

```powershell
docker run --rm -d --name mv-dq21-restore `
  -e MYSQL_ROOT_PASSWORD=<isolated-password> `
  -p 3307:3306 mysql:8.0.36
docker cp .local/dq21-backup.sql.gz mv-dq21-restore:/tmp/dq21-backup.sql.gz
docker exec mv-dq21-restore sh -c `
  'gunzip -c /tmp/dq21-backup.sql.gz | mysql -uroot -p"$MYSQL_ROOT_PASSWORD"'
```

Restore Verification：

1. 校验 Migration 版本表。
2. 比较 Run/Step/Attempt/Checkpoint/Recovery/Idempotency/Audit 行数。
3. 抽查 Checkpoint Hash、Schema、Runtime Version 与 AES-GCM 解密。
4. 清空恢复实例对应 Redis namespace，不允许复用旧 Lease。
5. 启动两个 Worker 竞争恢复；旧 Fence 写入必须失败。
6. 记录 RTO/RPO 实测值。

## 8. Failure Matrix

| Scenario | Expected | Current |
| --- | --- | --- |
| MySQL unavailable | No authoritative write | Unit fail-closed logic；Infra Not Verified |
| Redis unavailable | Lease operation fails safely | Redis script/error path code；Infra Not Verified |
| Lease expired | Old owner loses write authority | Unit/Candidate verified；Infra Not Verified |
| Worker crash | Recovery attempt path | Coordinator tests；Multi-process Not Verified |
| Owner failover | New fence | Test implementation exists；Not Verified |
| Stale owner write | DB rejects | Unit/Candidate verified；Real MySQL Not Verified |
| Concurrent write | One owner remains authoritative | DB conditional update code；Infra Not Verified |
| Checkpoint corrupted | Reject recovery | Unit verified |
| Recovery failure | Failure recorded | Unit verified |
| Cleanup failure | Job failed + retry next batch | Unit verified |
| Backup failure | No success evidence | Not Verified |
| Restore failure | Isolated restore fails safely | Not Verified |
| Redis restart | Leases re-established | Not Verified |
| Migration failure | Goose failure handling | Integration test code；Not Verified |
