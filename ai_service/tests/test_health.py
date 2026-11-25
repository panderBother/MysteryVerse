from fastapi.testclient import TestClient

from mysteryverse_ai.config import Settings
from mysteryverse_ai.main import create_app


def test_health_endpoints() -> None:
    app = create_app(
        Settings.for_test().model_copy(update={"environment": "test", "log_level": "ERROR"})
    )
    client = TestClient(app)

    for path in ("/health/live", "/health/ready", "/internal/v1/health"):
        response = client.get(path)

        assert response.status_code == 200
        assert response.json()["data"]["service"] == "ai-api"
        assert response.headers["X-Request-Id"]
        assert response.json()["meta"]["trace_id"]


def test_request_id_is_preserved() -> None:
    app = create_app(
        Settings.for_test().model_copy(update={"environment": "test", "log_level": "ERROR"})
    )
    client = TestClient(app)

    response = client.get(
        "/health/live",
        headers={"X-Request-Id": "client-request-id", "X-Trace-Id": "client-trace-id"},
    )

    assert response.headers["X-Request-Id"] == "client-request-id"
    assert response.headers["X-Trace-Id"] == "client-trace-id"
    assert response.json()["meta"]["request_id"] == "client-request-id"


def test_not_found_uses_error_envelope() -> None:
    app = create_app(
        Settings.for_test().model_copy(update={"environment": "test", "log_level": "ERROR"})
    )
    client = TestClient(app)

    response = client.get("/missing")

    assert response.status_code == 404
    assert response.json()["error"]["code"] == "HTTP_404"
    assert response.json()["error"]["request_id"]
