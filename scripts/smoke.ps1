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
    $requestBody = Get-Content -LiteralPath $requestPath -Raw
    $requestJson = $requestBody | ConvertFrom-Json
    $runId = [string]$requestJson.run_id
    $headers = @{
        "X-Service-Id" = $(if ($env:AI_INTERNAL_SERVICE_ID) { $env:AI_INTERNAL_SERVICE_ID } else { "go-core" })
        "X-Service-Token" = $aiServiceToken
        "X-Service-Scope" = "ai:agent:run,run:$runId"
        "X-Request-Id" = "agent-runtime-request-01"
        "X-Trace-Id" = "01J9Z3K4M7Q2R8T5V6W1XT0000"
        "Idempotency-Key" = "agent-runtime-smoke-01"
    }
    if ($env:AI_INTERNAL_REQUEST_SIGNING_KEY) {
        $timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds().ToString()
        $nonceBytes = [Security.Cryptography.RandomNumberGenerator]::GetBytes(16)
        $nonce = [Convert]::ToHexString($nonceBytes).ToLowerInvariant()
        $keyId = $(if ($env:AI_INTERNAL_SIGNING_KEY_ID) {
                $env:AI_INTERNAL_SIGNING_KEY_ID
            }
            else {
                "dev-go-core-v1"
            })
        $bodyBytes = [Text.Encoding]::UTF8.GetBytes($requestBody)
        $bodyHash = [Convert]::ToHexString(
            [Security.Cryptography.SHA256]::HashData($bodyBytes)
        ).ToLowerInvariant()
        $canonical = @(
            "mv-hmac-sha256-v1",
            $keyId,
            $timestamp,
            $nonce,
            $headers["X-Service-Id"],
            "POST",
            "/internal/v1/ai/agent-runs",
            "",
            $bodyHash
        ) -join "`n"
        $hmac = [Security.Cryptography.HMACSHA256]::new(
            [Text.Encoding]::UTF8.GetBytes($env:AI_INTERNAL_REQUEST_SIGNING_KEY)
        )
        try {
            $signature = [Convert]::ToHexString(
                $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($canonical))
            ).ToLowerInvariant()
        }
        finally {
            $hmac.Dispose()
        }
        $headers["X-Request-Signature-Version"] = "mv-hmac-sha256-v1"
        $headers["X-Service-Key-Id"] = $keyId
        $headers["X-Request-Timestamp"] = $timestamp
        $headers["X-Request-Nonce"] = $nonce
        $headers["X-Request-Signature"] = $signature
    }
    $agentResponse = Invoke-RestMethod `
        -Uri "$AiBaseUrl/internal/v1/ai/agent-runs" `
        -Method Post `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $requestBody `
        -TimeoutSec 10
    if ($agentResponse.status -ne "succeeded") {
        throw "AI agent smoke failed with status $($agentResponse.status)"
    }
    Write-Host "AI agent runtime: $($agentResponse.status)" -ForegroundColor Green
}

Write-Host "Smoke checks passed." -ForegroundColor Green
