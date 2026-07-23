import pytest

from app.core.config.settings import get_settings


@pytest.fixture(autouse=True)
def _env(monkeypatch):
    monkeypatch.setenv("AUTH_SECRET", "test-secret")
    monkeypatch.setenv("POSTGRES_DB_HOST", "localhost")
    monkeypatch.setenv("POSTGRES_DB_PORT", "5432")
    monkeypatch.setenv("POSTGRES_DB_LOGIN", "u")
    monkeypatch.setenv("POSTGRES_DB_PASSWORD", "p")
    monkeypatch.setenv("POSTGRES_DB_NAME", "db")
    monkeypatch.setenv("SAMPLER_INTERVAL", "30")
    get_settings.cache_clear()
    yield
    get_settings.cache_clear()
