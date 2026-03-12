import logging
import os

from app.api import register_routers
from app.repositories.providers import (
    provide_snapshot_repository,
    provide_snapshot_repository_stub,
)
from app.repositories.snapshot import SnapshotRepository
from app.services.providers import (
    provide_room_manager,
    provide_room_manager_stub,
    provide_sampler,
    provide_sampler_stub,
)
from app.services.room import RoomManager
from app.services.sampler import Sampler
from asyncpg import Pool
from fastapi import FastAPI


class Application:
    def __init__(self, app: FastAPI, pool: Pool) -> None:
        self.app = app
        self.pool = pool
        self.room_manager: RoomManager | None = None
        self.logger = logging.getLogger(self.__class__.__name__)

    def _configure_logging(self) -> None:
        logging.basicConfig(
            level=int(os.environ.get("LOGGING_LEVEL", logging.DEBUG)),
            format="%(levelname)s %(asctime)s %(filename)s:%(lineno)d %(message)s",
        )

    def _create_repositories(self) -> None:
        self.snapshot_repository = provide_snapshot_repository(self.pool)

    def _create_services(self) -> None:
        self.room_manager = provide_room_manager()
        self.sampler = provide_sampler(self.snapshot_repository)

    def _override_dependencies(self) -> None:
        self.app.dependency_overrides = {
            provide_snapshot_repository_stub: lambda: self.snapshot_repository,
            provide_room_manager_stub: lambda: self.room_manager,
            provide_sampler_stub: lambda: self.sampler,
        }

    def _add_routes(self) -> None:
        register_routers(self.app)

    def build(self) -> "Application":
        self._configure_logging()
        self._create_repositories()
        self._create_services()
        self._override_dependencies()
        self._add_routes()
        return self
