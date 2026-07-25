import base64

import pytest

from app.platform import crypto
from app.platform.config.settings import get_settings


@pytest.fixture(autouse=True)
def _env(monkeypatch):
    monkeypatch.setenv("AUTH_SECRET", "test-secret")
    monkeypatch.setenv("PRESENCE_KEY", base64.b64encode(b"0123456789abcdef0123456789abcdef").decode())
    monkeypatch.setenv("POSTGRES_DB_HOST", "localhost")
    monkeypatch.setenv("POSTGRES_DB_PORT", "5432")
    monkeypatch.setenv("POSTGRES_DB_LOGIN", "u")
    monkeypatch.setenv("POSTGRES_DB_PASSWORD", "p")
    monkeypatch.setenv("POSTGRES_DB_NAME", "db")
    monkeypatch.setenv("SAMPLER_INTERVAL", "30")
    get_settings.cache_clear()
    crypto._cipher.cache_clear()
    yield
    get_settings.cache_clear()
    crypto._cipher.cache_clear()
