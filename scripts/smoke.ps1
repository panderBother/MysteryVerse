param(
    [string]$GoBaseUrl = $(if ($env:GO_SMOKE_BASE_URL) { $env:GO_SMOKE_BASE_URL } else { "http://127.0.0.1:8080" }),
    [string]$AiBaseUrl = $(if ($env:AI_SMOKE_BASE_URL) { $env:AI_SMOKE_BASE_URL } else { "http://127.0.0.1:8000" }),
    [string]$WebBaseUrl = $(if ($env:WEB_SMOKE_BASE_URL) { $env:WEB_SMOKE_BASE_URL } else { "http://127.0.0.1:5173" })
)

$ErrorActionPreference = "Stop"

$checks = @(
    @{ Name = "Go live"; Url = "$GoBaseUrl/health/live" },
    @{ Name = "Go ready"; Url = "$GoBaseUrl/health/ready" },
    @{ Name = "Go API health"; Url = "$GoBaseUrl/api/v1/health" },
    @{ Name = "AI live"; Url = "$AiBaseUrl/health/live" },
    @{ Name = "AI ready"; Url = "$AiBaseUrl/health/ready" },
    @{ Name = "Web"; Url = $WebBaseUrl }
)

foreach ($check in $checks) {
    $response = Invoke-WebRequest -Uri $check.Url -UseBasicParsing -TimeoutSec 10
    if ($response.StatusCode -lt 200 -or $response.StatusCode -ge 300) {
        throw "$($check.Name) failed with status $($response.StatusCode)"
    }
    Write-Host "$($check.Name): $($response.StatusCode)" -ForegroundColor Green
}

$aiServiceToken = $env:AI_INTERNAL_SERVICE_TOKEN
if ($aiServiceToken) {
    $requestPath = Join-Path $PSScriptRoot "..\packages\contracts\examples\agent-runtime-request.json"
    $headers = @{
        "X-Service-Id" = $(if ($env:AI_INTERNAL_SERVICE_ID) { $env:AI_INTERNAL_SERVICE_ID } else { "go-core" })
        "X-Service-Token" = $aiServiceToken
        "X-Request-Id" = "agent-runtime-request-01"
        "X-Trace-Id" = "01J9Z3K4M7Q2R8T5V6W1XT0000"
        "Idempotency-Key" = "agent-runtime-smoke-01"
    }
    $agentResponse = Invoke-RestMethod `
        -Uri "$AiBaseUrl/internal/v1/ai/agent-runs" `
        -Method Post `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (Get-Content -LiteralPath $requestPath -Raw) `
        -TimeoutSec 10
    if ($agentResponse.status -ne "succeeded") {
        throw "AI agent smoke failed with status $($agentResponse.status)"
    }
    Write-Host "AI agent runtime: $($agentResponse.status)" -ForegroundColor Green
}

Write-Host "Smoke checks passed." -ForegroundColor Green
