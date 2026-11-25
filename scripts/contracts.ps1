[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet("generate", "validate", "check", "test")]
    [string]$Command = "check"
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$GeneratedPaths = @(
    "apps/web/src/generated/contracts.ts",
    "backend/internal/contracts/contracts_gen.go",
    "ai_service/src/mysteryverse_ai/schemas/generated/models.py",
    "ai_service/src/mysteryverse_ai/schemas/generated/active_output_schemas.py",
    "ai_service/src/mysteryverse_ai/schemas/generated/schema-registry.json",
    "packages/contracts/compatibility/schema-fingerprint.json",
    "packages/contracts/compatibility/schema-baseline-v1.json",
    "packages/contracts/registry/schema-registry.json"
)

function Invoke-Generate {
    Push-Location $Root
    try {
        & uv run --project packages/contracts/tools python packages/contracts/scripts/generate.py
        if ($LASTEXITCODE -ne 0) {
            throw "contract generation failed"
        }
    }
    finally {
        Pop-Location
    }
}

function Invoke-Validate {
    Push-Location $Root
    try {
        & uv run --project packages/contracts/tools python packages/contracts/scripts/validate.py
        if ($LASTEXITCODE -ne 0) {
            throw "contract validation failed"
        }
    }
    finally {
        Pop-Location
    }
}

function Invoke-ContractTests {
    $backendPath = Join-Path $Root "backend"
    Push-Location $backendPath
    try {
        & go test ./internal/contracts
        if ($LASTEXITCODE -ne 0) {
            throw "Go contract tests failed"
        }
    }
    finally {
        Pop-Location
    }

    Push-Location $Root
    try {
        & uv run --project ai_service pytest packages/contracts/tests ai_service/tests/test_contracts.py
        if ($LASTEXITCODE -ne 0) {
            throw "Python contract tests failed"
        }

        & (Get-Command npm.cmd).Source --workspace @mysteryverse/web run test:run -- tests/contracts.test.ts
        if ($LASTEXITCODE -ne 0) {
            throw "TypeScript contract tests failed"
        }
    }
    finally {
        Pop-Location
    }
}

function Invoke-Check {
    $snapshot = Join-Path $Root ".local/contracts-snapshot"
    if (Test-Path -LiteralPath $snapshot) {
        $resolvedSnapshot = [System.IO.Path]::GetFullPath($snapshot)
        $resolvedRoot = [System.IO.Path]::GetFullPath($Root)
        if (-not $resolvedSnapshot.StartsWith($resolvedRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
            throw "Refusing to remove snapshot outside repository"
        }
        Remove-Item -LiteralPath $snapshot -Recurse -Force
    }

    New-Item -ItemType Directory -Path $snapshot -Force | Out-Null
    foreach ($relativePath in $GeneratedPaths) {
        $source = Join-Path $Root $relativePath
        if (Test-Path -LiteralPath $source) {
            $destination = Join-Path $snapshot $relativePath
            New-Item -ItemType Directory -Path (Split-Path -Parent $destination) -Force | Out-Null
            Copy-Item -LiteralPath $source -Destination $destination
        }
    }

    Invoke-Generate
    Invoke-Validate

    foreach ($relativePath in $GeneratedPaths) {
        $expected = Join-Path $snapshot $relativePath
        $actual = Join-Path $Root $relativePath
        if (-not (Test-Path -LiteralPath $expected)) {
            continue
        }
        $expectedHash = (Get-FileHash -LiteralPath $expected -Algorithm SHA256).Hash
        $actualHash = (Get-FileHash -LiteralPath $actual -Algorithm SHA256).Hash
        if ($expectedHash -ne $actualHash) {
            throw "generated artifact drift detected: $relativePath"
        }
    }

    Invoke-ContractTests
    Write-Host "Contracts check passed." -ForegroundColor Green
}

switch ($Command) {
    "generate" { Invoke-Generate }
    "validate" { Invoke-Validate }
    "check" { Invoke-Check }
    "test" { Invoke-ContractTests }
}
