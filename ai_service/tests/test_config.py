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


def test_signing_key_requires_key_id_and_minimum_length() -> None:
    with pytest.raises(ValidationError):
        Settings.model_validate(
            {
                "internal_signing_key_id": "",
                "internal_request_signing_key": "a" * 32,
            }
        )
    with pytest.raises(ValidationError):
        Settings.model_validate(
            {
                "internal_signing_key_id": "go-core-v1",
                "internal_request_signing_key": "short",
            }
        )


def test_production_internal_security_is_fail_closed() -> None:
    with pytest.raises(ValidationError):
        Settings.model_validate(
            {
                "environment": "production",
                "internal_service_token": "token",
                "internal_signing_key_id": "go-core-v1",
                "internal_request_signing_key": "a" * 32,
            }
        )


def test_runtime_persistence_requires_database_and_encryption_key() -> None:
    with pytest.raises(ValidationError):
        Settings.model_validate({"runtime_persistence_enabled": True})
    with pytest.raises(ValidationError):
        Settings.model_validate(
            {
                "runtime_persistence_enabled": True,
                "database_url": "mysql://user:pass@localhost/db",
            }
        )
    settings = Settings.model_validate(
        {
            "runtime_persistence_enabled": True,
            "database_url": "mysql://user:pass@localhost/db",
            "checkpoint_encryption_key": "k" * 32,
        }
    )
    assert settings.runtime_persistence_enabled is True
