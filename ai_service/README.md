# MysteryVerse AI Service

Python AI API, generic Agent Runtime and Worker foundation.

```powershell
uv sync --project ai_service
uv run --project ai_service python -m mysteryverse_ai.main
```

The service exposes:

```text
GET  /health/live
GET  /health/ready
GET  /internal/v1/health
POST /internal/v1/ai/agent-runs
```

The internal Agent endpoint requires `X-Service-Id`, `X-Service-Token`,
`X-Request-Id`, `X-Trace-Id` and `Idempotency-Key`.

The default `mock` provider requires no API key. To use the real HTTP provider:

```powershell
$env:AI_MODEL_PROVIDER = "openai_compatible"
$env:AI_MODEL_NAME = "<model-name>"
$env:AI_MODEL_BASE_URL = "https://api.openai.com/v1"
$env:AI_MODEL_API_KEY = "<provided-at-runtime>"
$env:AI_INTERNAL_SERVICE_TOKEN = "<provided-at-runtime>"
```

Do not place real credentials in `.env.example` or source control.

The implemented demo graph performs a finite model -> tool -> model ->
structured-output validation loop. AI DM, NPC behavior, MemoryCandidate
acceptance and game-state mutation are intentionally not implemented.

Optional request integrity prototype:

```text
X-Request-Timestamp
X-Request-Nonce
X-Request-Signature
```

Set `AI_INTERNAL_REQUEST_SIGNING_KEY` to require HMAC-SHA256 signed requests.
`AI_INTERNAL_REQUEST_SIGNING_PREVIOUS_KEY` supports a rotation overlap. D3/D4
context messages are excluded from model context unless
`AI_ALLOW_RESTRICTED_MODEL_CONTEXT=true`; this is not approval to send private
script content.
