[CmdletBinding()]
param(
    [switch]$ResetTestDatabase
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$InfraScript = Join-Path $PSScriptRoot "dq21-infra.ps1"

function Import-DotEnv {
    $envPath = Join-Path $Root ".env"
    if (-not (Test-Path -LiteralPath $envPath)) {
        return
    }
    foreach ($line in Get-Content -LiteralPath $envPath) {
        $trimmed = $line.Trim()
        if ($trimmed.Length -eq 0 -or $trimmed.StartsWith("#")) {
            continue
        }
        $separator = $trimmed.IndexOf("=")
        if ($separator -lt 1) {
            continue
        }
        $key = $trimmed.Substring(0, $separator).Trim()
        $value = $trimmed.Substring($separator + 1).Trim()
        [Environment]::SetEnvironmentVariable($key, $value, "Process")
    }
}

Import-DotEnv

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw @"
Docker is unavailable.
DQ-21 Integration, Multi-Worker and Backup/Restore verification remain NOT VERIFIED.
Install Docker Desktop, copy .env.example to .env, then run:
  ./scripts/dq21-infra.ps1 up
  ./scripts/dq21-verify.ps1 -ResetTestDatabase
"@
}

& powershell -NoProfile -ExecutionPolicy Bypass -File $InfraScript check
if ($LASTEXITCODE -ne 0) {
    throw "DQ-21 infrastructure preflight failed."
}
& powershell -NoProfile -ExecutionPolicy Bypass -File $InfraScript up
if ($LASTEXITCODE -ne 0) {
    throw "DQ-21 infrastructure startup failed."
}
if ($ResetTestDatabase) {
    & powershell -NoProfile -ExecutionPolicy Bypass -File $InfraScript `
        reset-test-db -ConfirmTestDatabaseReset
    if ($LASTEXITCODE -ne 0) {
        throw "DQ-21 test database reset failed."
    }
}

$db = "mysteryverse_dq21_test"
$port = if ($env:MYSQL_PORT) { $env:MYSQL_PORT } else { "3306" }
$redisPort = if ($env:REDIS_PORT) { $env:REDIS_PORT } else { "6379" }
$rootDsn = "root:$($env:MYSQL_ROOT_PASSWORD)@tcp(127.0.0.1:$port)/$db?parseTime=true"
$asyncDsn = "mysql+asyncmy://root:$($env:MYSQL_ROOT_PASSWORD)@127.0.0.1:$port/$db"
$redisUrl = "redis://:$($env:REDIS_PASSWORD)@127.0.0.1:$redisPort/0"

Push-Location (Join-Path $Root "backend")
try {
    $env:MYSQL_MIGRATION_TEST_DSN = $rootDsn
    & go test ./migrations -run TestMigrationsUpStatusAndDown -count=1 -v
    if ($LASTEXITCODE -ne 0) {
        throw "Migration integration verification failed."
    }

    $env:MYSQL_DSN = $rootDsn
    & go run ./cmd/migrate --command=up
    if ($LASTEXITCODE -ne 0) {
        throw "Migration apply for runtime integration failed."
    }
}
finally {
    Pop-Location
}

Push-Location $Root
try {
    $env:AI_RUNTIME_TEST_DATABASE_URL = $asyncDsn
    $env:AI_RUNTIME_TEST_REDIS_URL = $redisUrl
    & uv run --project ai_service pytest `
        ai_service/tests/test_runtime_persistence_mysql.py `
        ai_service/tests/test_runtime_persistence_multiworker.py `
        -m integration -q
    if ($LASTEXITCODE -ne 0) {
        throw "MySQL/Redis runtime integration verification failed."
    }
    & uv run --project ai_service pytest ai_service/tests
    if ($LASTEXITCODE -ne 0) {
        throw "Regression test suite failed."
    }
}
finally {
    Pop-Location
}

Write-Host @"
DQ-21 automated integration checks completed.
Unit, Migration, MySQL/Redis Integration and two-process failover checks ran.
Backup/Restore still requires separate isolated-instance evidence:
  docs/architecture/task-13-verification.md
"@ -ForegroundColor Green
