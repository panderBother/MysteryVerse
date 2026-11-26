[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("check", "up", "down", "reset-test-db", "destroy")]
    [string]$Command = "check",
    [switch]$ConfirmTestDatabaseReset,
    [switch]$ConfirmDeleteVolumes
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot

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

function Assert-Docker {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        throw "Docker is not installed. DQ-21 real infrastructure verification remains NOT VERIFIED."
    }
}

function Invoke-Compose {
    param([string[]]$Arguments)

    Push-Location $Root
    try {
        & docker compose @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "docker compose failed: $($Arguments -join ' ')"
        }
    }
    finally {
        Pop-Location
    }
}

function Wait-Infrastructure {
    $deadline = (Get-Date).AddMinutes(2)
    while ((Get-Date) -lt $deadline) {
        & docker compose exec -T mysql mysqladmin ping -h 127.0.0.1 `
            -uroot -p"$env:MYSQL_ROOT_PASSWORD" --silent *> $null
        $mysqlReady = $LASTEXITCODE -eq 0
        & docker compose exec -T redis redis-cli -a "$env:REDIS_PASSWORD" ping *> $null
        $redisReady = $LASTEXITCODE -eq 0
        if ($mysqlReady -and $redisReady) {
            Write-Host "MySQL and Redis are healthy." -ForegroundColor Green
            return
        }
        Start-Sleep -Seconds 2
    }
    throw "MySQL/Redis did not become healthy within two minutes."
}

function Reset-TestDatabase {
    if (-not $ConfirmTestDatabaseReset) {
        throw "reset-test-db is destructive. Re-run with -ConfirmTestDatabaseReset."
    }
    $database = "mysteryverse_dq21_test"
    $sql = @"
DROP DATABASE IF EXISTS $database;
CREATE DATABASE $database CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE DATABASE IF NOT EXISTS mv_ai CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
GRANT ALL PRIVILEGES ON $database.* TO '$env:MYSQL_USER'@'%';
GRANT ALL PRIVILEGES ON mv_ai.* TO '$env:MYSQL_USER'@'%';
FLUSH PRIVILEGES;
"@
    $sql | docker compose exec -T mysql mysql -uroot -p"$env:MYSQL_ROOT_PASSWORD"
    if ($LASTEXITCODE -ne 0) {
        throw "failed to reset isolated DQ-21 test database"
    }
    Write-Host "Reset isolated test database: $database" -ForegroundColor Yellow
}

Import-DotEnv

switch ($Command) {
    "check" {
        Assert-Docker
        Invoke-Compose -Arguments @("config", "--quiet")
        Invoke-Compose -Arguments @("ps")
        Write-Host "DQ-21 infrastructure configuration is valid." -ForegroundColor Green
    }
    "up" {
        Assert-Docker
        Invoke-Compose -Arguments @("up", "-d", "mysql", "redis")
        Wait-Infrastructure
    }
    "down" {
        Assert-Docker
        Invoke-Compose -Arguments @("down")
    }
    "reset-test-db" {
        Assert-Docker
        Reset-TestDatabase
    }
    "destroy" {
        Assert-Docker
        if (-not $ConfirmDeleteVolumes) {
            throw "destroy deletes mysql_data and redis_data. Re-run with -ConfirmDeleteVolumes."
        }
        Invoke-Compose -Arguments @("down", "-v")
    }
}
