import asyncpg

from app.modules.chats.application.interfaces import IChatsUnitOfWork
from app.modules.chats.infrastructure.repositories import MessageRepository
from app.platform.db.uow import BaseUnitOfWork


class ChatsUnitOfWork(BaseUnitOfWork, IChatsUnitOfWork):
    def _bind(self, conn: asyncpg.Connection) -> None:
        self.messages = MessageRepository(conn)
