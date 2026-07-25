from abc import ABC, abstractmethod
from datetime import datetime
from types import TracebackType
from typing import Self

from app.modules.chats.application.dto import MessageDTO


class IMessageRepository(ABC):
    @abstractmethod
    async def add(self, room_token: str, from_account: str, to_account: str, text: str) -> datetime: ...


class IMessageReader(ABC):
    @abstractmethod
    async def history(self, room_token: str, account: str) -> list[MessageDTO]: ...


class IChatsUnitOfWork(ABC):
    messages: IMessageRepository

    @abstractmethod
    async def __aenter__(self) -> Self: ...

    @abstractmethod
    async def __aexit__(
        self, exc_type: type[BaseException] | None, exc: BaseException | None, tb: TracebackType | None
    ) -> None: ...


class IMessageDelivery(ABC):
    @abstractmethod
    async def deliver(
        self, room_token: str, from_account: str, from_name: str, to_account: str, text: str, at: str
    ) -> None: ...


class IRoomMembership(ABC):
    @abstractmethod
    async def is_member(self, room_token: str, account_id: str) -> bool: ...
