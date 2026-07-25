from abc import ABC, abstractmethod
from types import TracebackType
from typing import Self


class IMembershipRepository(ABC):
    @abstractmethod
    async def add_member(self, room_id: str, account_id: str, role: str = "member") -> None: ...

    @abstractmethod
    async def remove_member(self, room_id: str, account_id: str) -> None: ...


class IMembershipReader(ABC):
    @abstractmethod
    async def is_member(self, room_id: str, account_id: str) -> bool: ...

    @abstractmethod
    async def owned_room(self, account_id: str) -> str | None: ...

    @abstractmethod
    async def list_members(self, room_id: str) -> list[dict]: ...

    @abstractmethod
    async def role_of(self, room_id: str, account_id: str) -> str | None: ...


class IRoomsUnitOfWork(ABC):
    memberships: IMembershipRepository

    @abstractmethod
    async def __aenter__(self) -> Self: ...

    @abstractmethod
    async def __aexit__(
        self, exc_type: type[BaseException] | None, exc: BaseException | None, tb: TracebackType | None
    ) -> None: ...


class IInviteCodec(ABC):
    @abstractmethod
    def sign(self, room_id: str) -> str: ...

    @abstractmethod
    def verify(self, code: str) -> str | None: ...
