import logging

from app.platform.config import get_settings


def configure_logging() -> None:
    logging.basicConfig(
        level=get_settings().logging_level,
        format="%(levelname)s %(asctime)s %(filename)s:%(lineno)d %(message)s",
    )
