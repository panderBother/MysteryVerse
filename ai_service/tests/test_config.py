import pytest
from pydantic import ValidationError

from mysteryverse_ai.config import Settings


def test_settings_defaults() -> None:
    settings = Settings.for_test()

    assert settings.environment == "development"
    assert settings.port == 8000
    assert settings.log_level == "INFO"
    assert settings.model_provider == "mock"
    assert settings.agent_default_max_steps == 6
    assert settings.allow_restricted_model_context is False
    assert settings.internal_request_signing_key is None


def test_openai_provider_requires_api_key() -> None:
    with pytest.raises(ValidationError):
        Settings.model_validate({"model_provider": "openai_compatible"})
