[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet(
        "install",
        "dev",
        "build",
        "test",
        "lint",
        "format",
        "check",
        "docker-up",
        "docker-down",
        "smoke",
        "structure",
        "contracts-generate",
        "contracts-check",
        "db-generate",
        "db-check",
        "db-migrate",
        "db-status",
        "db-rollback",
        "db-test"
    )]
    [string]$Command = "check"
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

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$ArgumentList = @(),
        [string]$WorkingDirectory = $Root
    )

    $resolvedFilePath = if ($FilePath -eq "npm") {
        (Get-Command npm.cmd).Source
    }
    else {
        $FilePath
    }

    Write-Host "> $FilePath $($ArgumentList -join ' ')" -ForegroundColor Cyan
    Push-Location $WorkingDirectory
    try {
        & $resolvedFilePath @ArgumentList
        if ($LASTEXITCODE -ne 0) {
            throw "Command failed with exit code ${LASTEXITCODE}: $FilePath $($ArgumentList -join ' ')"
        }
    }
    finally {
        Pop-Location
    }
}

function Assert-Docker {
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
        throw "Docker CLI is not available. Install Docker Desktop/Engine and retry."
    }
}

function Invoke-Install {
    Invoke-Native -FilePath npm -ArgumentList @("install")
    Invoke-Native -FilePath go -ArgumentList @("mod", "download") -WorkingDirectory (Join-Path $Root "backend")
    Invoke-Native -FilePath uv -ArgumentList @("sync", "--project", "ai_service")
    Invoke-Native -FilePath uv -ArgumentList @("sync", "--project", "packages/contracts/tools")
}

function Invoke-Structure {
    Invoke-Native -FilePath powershell -ArgumentList @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $Root "scripts/check-structure.ps1")
    )
}

function Invoke-Lint {
    Invoke-Native -FilePath npm -ArgumentList @("run", "lint:web")
    Invoke-Native -FilePath npm -ArgumentList @("run", "typecheck:web")
    Invoke-Native -FilePath npm -ArgumentList @("run", "format:check:web")
    Invoke-Native -FilePath go -ArgumentList @("vet", "./...") -WorkingDirectory (Join-Path $Root "backend")
    Push-Location (Join-Path $Root "backend")
    try {
        $unformatted = & gofmt -l . 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "gofmt failed: $unformatted"
        }
        if ($unformatted) {
            throw "Go files need formatting:`n$($unformatted -join "`n")"
        }
    }
    finally {
        Pop-Location
    }
    Invoke-Native -FilePath uv -ArgumentList @("run", "--project", "ai_service", "ruff", "check", "ai_service")
    Invoke-Native -FilePath uv -ArgumentList @(
        "run",
        "--project",
        "ai_service",
        "ruff",
        "format",
        "--check",
        "ai_service"
    )
    Invoke-Native -FilePath uv -ArgumentList @("run", "--project", "ai_service", "mypy", "ai_service/src")
}

function Invoke-Test {
    Invoke-Native -FilePath npm -ArgumentList @("run", "test:web")
    Invoke-Native -FilePath go -ArgumentList @("test", "./...") -WorkingDirectory (Join-Path $Root "backend")
    Invoke-Native -FilePath uv -ArgumentList @(
        "run",
        "--project",
        "ai_service",
        "pytest",
        "ai_service/tests"
    )
}

function Invoke-Build {
    Invoke-Native -FilePath npm -ArgumentList @("run", "build:web")
    Invoke-Native -FilePath go -ArgumentList @(
        "build",
        "./cmd/api",
        "./cmd/realtime-runtime",
        "./cmd/worker",
        "./cmd/migrate"
    ) -WorkingDirectory (Join-Path $Root "backend")
    Invoke-Native -FilePath uv -ArgumentList @("build", "--project", "ai_service")
}

function Invoke-Format {
    Invoke-Native -FilePath npm -ArgumentList @("run", "format:web")
    Invoke-Native -FilePath gofmt -ArgumentList @("-w", ".") -WorkingDirectory (Join-Path $Root "backend")
    Invoke-Native -FilePath uv -ArgumentList @(
        "run",
        "--project",
        "ai_service",
        "ruff",
        "check",
        "--fix",
        "ai_service"
    )
    Invoke-Native -FilePath uv -ArgumentList @(
        "run",
        "--project",
        "ai_service",
        "ruff",
        "format",
        "ai_service"
    )
}

function Invoke-DockerUp {
    Assert-Docker
    Invoke-Native -FilePath docker -ArgumentList @("compose", "up", "-d")
    Invoke-Native -FilePath docker -ArgumentList @("compose", "ps")
}

function Invoke-DockerDown {
    Assert-Docker
    Invoke-Native -FilePath docker -ArgumentList @("compose", "down")
}

function Invoke-Smoke {
    Invoke-Native -FilePath powershell -ArgumentList @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $Root "scripts/smoke.ps1")
    )
}

function Invoke-Contracts {
    param([string]$ContractCommand)

    Invoke-Native -FilePath powershell -ArgumentList @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $Root "scripts/contracts.ps1"),
        $ContractCommand
    )
}

function Invoke-Database {
    param(
        [Parameter(Mandatory = $true)]
        [string]$DatabaseCommand,
        [string[]]$ArgumentList = @()
    )

    $arguments = @(
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        (Join-Path $Root "scripts/database.ps1"),
        $DatabaseCommand
    )
    $arguments += $ArgumentList

    Invoke-Native -FilePath powershell -ArgumentList $arguments
}

function Invoke-Dev {
    if ($env:MV_SKIP_INFRA -ne "1") {
        if (Get-Command docker -ErrorAction SilentlyContinue) {
            Invoke-DockerUp
        }
        else {
            Write-Warning "Docker is unavailable; starting application processes without MySQL/Redis."
        }
    }

    $started = @()
    try {
        $started += Start-Process -FilePath (Get-Command npm.cmd).Source `
            -ArgumentList @("run", "dev:web") `
            -WorkingDirectory $Root `
            -NoNewWindow `
            -PassThru

        $started += Start-Process -FilePath (Get-Command go.exe).Source `
            -ArgumentList @("run", "./cmd/api") `
            -WorkingDirectory (Join-Path $Root "backend") `
            -NoNewWindow `
            -PassThru

        $started += Start-Process -FilePath (Get-Command uv.exe).Source `
            -ArgumentList @("run", "--project", "ai_service", "python", "-m", "mysteryverse_ai.main") `
            -WorkingDirectory $Root `
            -NoNewWindow `
            -PassThru

        Write-Host "Development processes started. Press Ctrl+C to stop." -ForegroundColor Green
        while ($true) {
            Start-Sleep -Seconds 1
            foreach ($process in $started) {
                if ($process.HasExited) {
                    throw "Development process $($process.Id) exited with code $($process.ExitCode)."
                }
            }
        }
    }
    finally {
        foreach ($process in $started) {
            if (-not $process.HasExited) {
                Stop-Process -Id $process.Id -Force
            }
        }
    }
}

Import-DotEnv

switch ($Command) {
    "install" { Invoke-Install }
    "dev" { Invoke-Dev }
    "build" { Invoke-Build }
    "test" { Invoke-Test }
    "lint" { Invoke-Lint }
    "format" { Invoke-Format }
    "check" {
        Invoke-Structure
        Invoke-Contracts -ContractCommand "check"
        Invoke-Database -DatabaseCommand "check"
        Invoke-Lint
        Invoke-Test
        Invoke-Build
    }
    "docker-up" { Invoke-DockerUp }
    "docker-down" { Invoke-DockerDown }
    "smoke" { Invoke-Smoke }
    "structure" { Invoke-Structure }
    "contracts-generate" { Invoke-Contracts -ContractCommand "generate" }
    "contracts-check" { Invoke-Contracts -ContractCommand "check" }
    "db-generate" { Invoke-Database -DatabaseCommand "generate" }
    "db-check" { Invoke-Database -DatabaseCommand "check" }
    "db-migrate" { Invoke-Database -DatabaseCommand "migrate-up" }
    "db-status" { Invoke-Database -DatabaseCommand "migrate-status" }
    "db-rollback" { Invoke-Database -DatabaseCommand "migrate-down" }
    "db-test" { Invoke-Database -DatabaseCommand "test" }
}
