from abc import ABC, abstractmethod
from types import TracebackType
from typing import Self


class IAccountRepository(ABC):
    @abstractmethod
    async def create_account(self, account_id: str, secret_verifier: str) -> None: ...

    @abstractmethod
    async def create_device(self, device_id: str, account_id: str, name: str | None) -> None: ...

    @abstractmethod
    async def set_name(self, account_id: str, name: str) -> None: ...

    @abstractmethod
    async def revoke_device(self, account_id: str, device_id: str) -> bool: ...


class IAccountReader(ABC):
    @abstractmethod
    async def get_verifier(self, account_id: str) -> str | None: ...

    @abstractmethod
    async def is_device_active(self, account_id: str, device_id: str) -> bool: ...

    @abstractmethod
    async def name_of(self, account_id: str) -> str: ...

    @abstractmethod
    async def list_devices(self, account_id: str) -> list[dict]: ...


class IAccountsUnitOfWork(ABC):
    accounts: IAccountRepository

    @abstractmethod
    async def __aenter__(self) -> Self: ...

    @abstractmethod
    async def __aexit__(
        self, exc_type: type[BaseException] | None, exc: BaseException | None, tb: TracebackType | None
    ) -> None: ...


class IRoomDirectory(ABC):
    @abstractmethod
    async def create_room_for_owner(self, account_id: str) -> str: ...

    @abstractmethod
    async def owned_room(self, account_id: str) -> str | None: ...
