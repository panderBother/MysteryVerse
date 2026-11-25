[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet(
        "generate",
        "check",
        "migrate-up",
        "migrate-status",
        "migrate-down",
        "test"
    )]
    [string]$Command = "generate",
    [int64]$Version = -1
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$Backend = Join-Path $Root "backend"

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

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$ArgumentList = @(),
        [string]$WorkingDirectory = $Backend
    )

    Write-Host "> $FilePath $($ArgumentList -join ' ')" -ForegroundColor Cyan
    Push-Location $WorkingDirectory
    try {
        & $FilePath @ArgumentList
        if ($LASTEXITCODE -ne 0) {
            throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($ArgumentList -join ' ')"
        }
    }
    finally {
        Pop-Location
    }
}

function Invoke-Migrate {
    param([string]$MigrationCommand)

    $arguments = @(
        "run",
        "github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0",
        "generate",
        "-f",
        "sqlc.yaml"
    )
    if ($MigrationCommand -ne "generate") {
        $arguments = @(
            "run",
            "./cmd/migrate",
            "--command=$MigrationCommand"
        )
        if ($Version -ge 0) {
            $arguments += "--version=$Version"
        }
    }

    Invoke-Native -FilePath "go" -ArgumentList $arguments
}

function Get-SqlcFingerprint {
    $generatedDirectory = Join-Path $Backend "internal/storage/sqlc"
    if (-not (Test-Path -LiteralPath $generatedDirectory)) {
        throw "Generated sqlc directory does not exist: $generatedDirectory"
    }

    $lines = Get-ChildItem -LiteralPath $generatedDirectory -File |
        Sort-Object Name |
        ForEach-Object {
            $hash = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash
            "$($_.Name):$hash"
        }

    $payload = [Text.Encoding]::UTF8.GetBytes(($lines -join "`n"))
    $sha = [Security.Cryptography.SHA256]::Create()
    try {
        return -join ($sha.ComputeHash($payload) | ForEach-Object { $_.ToString("x2") })
    }
    finally {
        $sha.Dispose()
    }
}

Import-DotEnv

switch ($Command) {
    "generate" {
        Invoke-Native -FilePath "go" -ArgumentList @(
            "run",
            "github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0",
            "generate",
            "-f",
            "sqlc.yaml"
        )
    }
    "check" {
        $before = Get-SqlcFingerprint
        Invoke-Native -FilePath "go" -ArgumentList @(
            "run",
            "github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0",
            "generate",
            "-f",
            "sqlc.yaml"
        )
        $after = Get-SqlcFingerprint
        if ($before -ne $after) {
            throw "Generated sqlc code was stale. Review and commit the regenerated files."
        }
        Write-Host "sqlc generated code is current." -ForegroundColor Green
    }
    "migrate-up" { Invoke-Migrate -MigrationCommand "up" }
    "migrate-status" { Invoke-Migrate -MigrationCommand "status" }
    "migrate-down" { Invoke-Migrate -MigrationCommand "down" }
    "test" {
        foreach ($name in @("MYSQL_TEST_DSN", "MYSQL_MIGRATION_TEST_DSN")) {
            if (-not [Environment]::GetEnvironmentVariable($name, "Process")) {
                throw "$name is required for database integration tests."
            }
        }

        Invoke-Native -FilePath "go" -ArgumentList @(
            "test",
            "./migrations",
            "./internal/storage",
            "./internal/identity",
            "./internal/room",
            "./internal/script",
            "./internal/game/...",
            "-count=1",
            "-v"
        )
    }
}
