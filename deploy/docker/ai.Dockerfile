FROM python:3.12-slim

COPY --from=ghcr.io/astral-sh/uv:0.9.26 /uv /uvx /bin/

WORKDIR /app

COPY ai_service/pyproject.toml ai_service/uv.lock ./ai_service/
RUN uv sync --project ai_service --frozen --no-dev

COPY ai_service ./ai_service

WORKDIR /app/ai_service
EXPOSE 8000

CMD ["uv", "run", "--project", ".", "python", "-m", "mysteryverse_ai.main"]
