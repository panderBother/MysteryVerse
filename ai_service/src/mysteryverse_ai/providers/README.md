# Providers

`models.py` defines the provider-neutral async generation and streaming protocol.

- `MockProvider` is deterministic and is the default test/local provider.
- `OpenAICompatibleProvider` uses `httpx` against an OpenAI-compatible
  `/chat/completions` endpoint.
- `RetryingProvider` applies bounded retry and backoff policy.

Provider credentials are loaded only from Settings and are never logged.
