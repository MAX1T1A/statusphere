import logging
import os

from app.api import register_routers
from app.providers import provide_room_manager_stub
from app.room import RoomManager
from fastapi import FastAPI


class Application:
    def __init__(self, app: FastAPI) -> None:
        self.app = app
        self.room_manager: RoomManager | None = None
        self.logger = logging.getLogger(self.__class__.__name__)

    def _configure_logging(self) -> None:
        logging.basicConfig(
            level=int(os.environ.get("LOGGING_LEVEL", logging.DEBUG)),
            format="%(levelname)s %(asctime)s %(filename)s:%(lineno)d %(message)s",
        )

    def _create_room_manager(self) -> None:
        self.room_manager = RoomManager()

    def _override_dependencies(self) -> None:
        self.app.dependency_overrides = {
            provide_room_manager_stub: lambda: self.room_manager,
        }

    def _add_routes(self) -> None:
        register_routers(self.app)

    def build(self) -> "Application":
        self._configure_logging()
        self._create_room_manager()
        self._override_dependencies()
        self._add_routes()
        return self
