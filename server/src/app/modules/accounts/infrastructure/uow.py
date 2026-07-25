import asyncpg

from app.modules.accounts.application.interfaces import IAccountsUnitOfWork
from app.modules.accounts.infrastructure.repositories import AccountRepository
from app.platform.db.uow import BaseUnitOfWork


class AccountsUnitOfWork(BaseUnitOfWork, IAccountsUnitOfWork):
    def _bind(self, conn: asyncpg.Connection) -> None:
        self.accounts = AccountRepository(conn)
