import asyncpg
from app.modules.rooms.application.interfaces import IRoomsUnitOfWork
from app.modules.rooms.infrastructure.repositories import MembershipRepository
from app.platform.db.uow import BaseUnitOfWork


class RoomsUnitOfWork(BaseUnitOfWork, IRoomsUnitOfWork):
    def _bind(self, conn: asyncpg.Connection) -> None:
        self.memberships = MembershipRepository(conn)
