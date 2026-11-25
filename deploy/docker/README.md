# Docker

`ai.Dockerfile` builds the Python AI API with `uv` and the locked dependency set.

```powershell
docker build -f deploy/docker/ai.Dockerfile -t mysteryverse-ai .
```

The service must receive `AI_INTERNAL_SERVICE_TOKEN` and, for a real model,
`AI_MODEL_API_KEY` from runtime secrets. API keys must not be committed.
