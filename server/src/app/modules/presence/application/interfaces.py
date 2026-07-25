from abc import ABC, abstractmethod
from datetime import date


class ISnapshotReader(ABC):
    @abstractmethod
    async def summary(self, room_token: str, device_id: str, since: date) -> list[dict]: ...

    @abstractmethod
    async def spotify_aggregate(self, room_token: str, device_id: str, since: date) -> dict: ...


class ISnapshotWriter(ABC):
    @abstractmethod
    async def save_batch(self, rows: list[tuple[str, str, str, dict]]) -> None: ...


class ISampler(ABC):
    @abstractmethod
    async def put(self, room_token: str, account_id: str, device_id: str, data: dict) -> None: ...


class IPresenceBroadcast(ABC):
    @abstractmethod
    async def publish(
        self, room_token: str, account_id: str, account_name: str, device_id: str, data: dict
    ) -> None: ...
