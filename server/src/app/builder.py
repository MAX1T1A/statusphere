import logging

from app.api import register_routers
from app.core.config import get_settings
from app.repositories.providers import (
    provide_account_repository,
    provide_account_repository_stub,
    provide_membership_repository,
    provide_membership_repository_stub,
    provide_snapshot_repository,
    provide_snapshot_repository_stub,
)
from app.services.providers import (
    provide_account_service,
    provide_account_service_stub,
    provide_membership_service,
    provide_membership_service_stub,
    provide_room_manager,
    provide_room_manager_stub,
    provide_sampler,
    provide_sampler_stub,
    provide_snapshot_service,
    provide_snapshot_service_stub,
)
from asyncpg import Pool
from fastapi import FastAPI


class Application:
    def __init__(self, app: FastAPI, pool: Pool) -> None:
        self.app = app
        self.pool = pool

        self.logger = logging.getLogger(self.__class__.__name__)

    def _configure_logging(self) -> None:
        logging.basicConfig(
            level=get_settings().logging_level,
            format="%(levelname)s %(asctime)s %(filename)s:%(lineno)d %(message)s",
        )

    def _create_repositories(self) -> None:
        self.snapshot_repository = provide_snapshot_repository(self.pool)
        self.account_repository = provide_account_repository(self.pool)
        self.membership_repository = provide_membership_repository(self.pool)

    def _create_services(self) -> None:
        self.room_manager = provide_room_manager()
        self.sampler = provide_sampler(self.snapshot_repository)
        self.snapshot_service = provide_snapshot_service(self.snapshot_repository)
        self.account_service = provide_account_service(self.account_repository, self.membership_repository)
        self.membership_service = provide_membership_service(self.membership_repository)

    def _override_dependencies(self) -> None:
        self.app.dependency_overrides = {
            provide_snapshot_repository_stub: lambda: self.snapshot_repository,
            provide_account_repository_stub: lambda: self.account_repository,
            provide_membership_repository_stub: lambda: self.membership_repository,
            provide_room_manager_stub: lambda: self.room_manager,
            provide_sampler_stub: lambda: self.sampler,
            provide_snapshot_service_stub: lambda: self.snapshot_service,
            provide_account_service_stub: lambda: self.account_service,
            provide_membership_service_stub: lambda: self.membership_service,
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
