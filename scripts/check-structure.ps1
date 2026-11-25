param()

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot

$requiredPaths = @(
    "AGENTS.md",
    "README.md",
    "Makefile",
    ".env.example",
    ".node-version",
    "docker-compose.yml",
    "deploy/docker/ai.Dockerfile",
    "apps/web/package.json",
    "apps/web/src/app/router.tsx",
    "apps/web/src/api/http.ts",
    "apps/web/src/features",
    "apps/web/src/stores",
    "apps/web/tests",
    "backend/go.mod",
    "backend/cmd/api/main.go",
    "backend/cmd/realtime-runtime/main.go",
    "backend/cmd/worker/main.go",
    "backend/internal/platform",
    "backend/internal/ai/client",
    "backend/internal/script",
    "backend/internal/game",
    "backend/pkg/version/version.go",
    "ai_service/pyproject.toml",
    "ai_service/.python-version",
    "ai_service/src/mysteryverse_ai/main.py",
    "ai_service/src/mysteryverse_ai/api/routes",
    "ai_service/src/mysteryverse_ai/providers",
    "ai_service/src/mysteryverse_ai/agent_runtime",
    "ai_service/src/mysteryverse_ai/tools",
    "ai_service/src/mysteryverse_ai/context",
    "ai_service/src/mysteryverse_ai/memory",
    "ai_service/src/mysteryverse_ai/observability",
    "ai_service/src/mysteryverse_ai/workflows/generation",
    "ai_service/src/mysteryverse_ai/workers",
    "ai_service/tests",
    "packages/contracts/README.md",
    "packages/contracts/codegen.yaml",
    "packages/contracts/json-schema/mysteryverse.schema.json",
    "packages/contracts/examples/manifest.json",
    "packages/contracts/tools/pyproject.toml",
    "packages/contracts/scripts/generate.py",
    "packages/contracts/scripts/validate.py",
    "apps/web/src/generated/contracts.ts",
    "backend/internal/contracts/contracts_gen.go",
    "ai_service/src/mysteryverse_ai/schemas/generated/models.py",
    "packages/contracts/openapi",
    "packages/contracts/json-schema/game-package",
    "packages/contracts/json-schema/game-events",
    "packages/contracts/json-schema/ws-messages",
    "packages/contracts/json-schema/ai-outputs",
    "deploy/compose",
    "scripts/mv.ps1",
    "docs/architecture/gate-ai-checklist.md",
    "docs/architecture/dq-18-22-ai-runtime-design.md",
    "docs/architecture/ai-runtime-schema-inventory.md",
    "docs/architecture/task-11-approval-blockers.md",
    "docs/MysteryVerse-PRD.md"
)

$missing = foreach ($relativePath in $requiredPaths) {
    $path = Join-Path $Root $relativePath
    if (-not (Test-Path -LiteralPath $path)) {
        $relativePath
    }
}

if ($missing) {
    throw "Missing required paths:`n$($missing -join "`n")"
}

$unexpectedPrd = Join-Path $Root "docs/PRD.md"
if (Test-Path -LiteralPath $unexpectedPrd) {
    throw "Duplicate docs/PRD.md is not allowed. Use docs/MysteryVerse-PRD.md."
}

Write-Host "Monorepo structure check passed." -ForegroundColor Green
