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
`X-Service-Scope`, `X-Request-Id`, `X-Trace-Id` and `Idempotency-Key`.
The current scope set is:

```text
ai:agent:run,run:<run_id>
```

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

DQ-19 request integrity profile:

```text
X-Request-Signature-Version
X-Service-Key-Id
X-Request-Timestamp
X-Request-Nonce
X-Request-Signature
```

Set `AI_INTERNAL_REQUEST_SIGNING_KEY` and `AI_INTERNAL_SIGNING_KEY_ID` to require
`mv-hmac-sha256-v1` signed requests. Previous Key ID/key support a rotation
window; `AI_INTERNAL_REVOKED_KEY_IDS` revokes IDs. Development nonce storage is
in-process; `AI_INTERNAL_NONCE_BACKEND=redis` requires `AI_REDIS_URL` and uses
atomic `SET NX EX`. Nonce Store or Key Provider failure is fail-closed.

`AI_INTERNAL_MTLS_ENABLED=true` configures Uvicorn server TLS with client
certificate verification. Production startup requires mTLS, HMAC, Redis nonce
storage and an external KMS provider; real CA/KMS/Redis deployment is currently
`NOT VERIFIED`.

Generated output schemas come from the Contracts Registry. The only Active
output is `demo_result -> DemoAgentOutput@1`; model output cannot choose a
Schema. D3/D4 context messages are excluded from model context unless
`AI_ALLOW_RESTRICTED_MODEL_CONTEXT=true`; this is not approval to send private
script content.
