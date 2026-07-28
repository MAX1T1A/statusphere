import base64
import logging
import os
from dataclasses import dataclass
from functools import lru_cache

from app.platform.config.postgres import PostgresConfig

_LEVELS = {
    "CRITICAL": logging.CRITICAL,
    "ERROR": logging.ERROR,
    "WARNING": logging.WARNING,
    "INFO": logging.INFO,
    "DEBUG": logging.DEBUG,
    "NOTSET": logging.NOTSET,
}


def _resolve_level(raw: str) -> int:
    raw = raw.strip()
    if raw.isdigit():
        return int(raw)
    return _LEVELS.get(raw.upper(), logging.INFO)


@dataclass(frozen=True)
class Settings:
    postgres: PostgresConfig
    auth_secret: str
    presence_key: bytes
    sampler_interval: int
    logging_level: int
    photos_dir: str
    photo_expiry_minutes: int


def _load_presence_key() -> bytes:
    raw = os.environ.get("PRESENCE_KEY", "")
    if not raw:
        raise RuntimeError("PRESENCE_KEY env is not set")
    key = base64.b64decode(raw)
    if len(key) not in (16, 24, 32):
        raise RuntimeError("PRESENCE_KEY must be base64 of 16, 24 or 32 bytes")
    return key


@lru_cache
def get_settings() -> Settings:
    auth_secret = os.environ.get("AUTH_SECRET", "")
    if not auth_secret:
        raise RuntimeError("AUTH_SECRET env is not set")

    return Settings(
        presence_key=_load_presence_key(),
        postgres=PostgresConfig(
            host=os.environ["POSTGRES_DB_HOST"],
            port=int(os.environ["POSTGRES_DB_PORT"]),
            username=os.environ["POSTGRES_DB_LOGIN"],
            password=os.environ["POSTGRES_DB_PASSWORD"],
            dbname=os.environ["POSTGRES_DB_NAME"],
            pool_size=int(os.environ.get("POSTGRES_POOL_SIZE", 10)),
        ),
        auth_secret=auth_secret,
        sampler_interval=int(os.environ.get("SAMPLER_INTERVAL", 30)),
        logging_level=_resolve_level(os.environ.get("LOGGING_LEVEL", "INFO")),
        photos_dir=os.environ.get("PHOTOS_DIR", "/data/photos"),
        photo_expiry_minutes=int(os.environ.get("PHOTO_EXPIRY_MINUTES", 180)),
    )
