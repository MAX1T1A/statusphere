import logging
import os
from dataclasses import dataclass
from functools import lru_cache

from app.core.config.postgres import PostgresConfig

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
    sampler_interval: int
    logging_level: int


@lru_cache
def get_settings() -> Settings:
    auth_secret = os.environ.get("AUTH_SECRET", "")
    if not auth_secret:
        raise RuntimeError("AUTH_SECRET env is not set")

    return Settings(
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
    )
