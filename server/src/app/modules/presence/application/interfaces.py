from abc import ABC, abstractmethod
from datetime import date


class ISnapshotReader(ABC):
    @abstractmethod
    async def summary(self, room_token: str, device_id: str, since: date) -> list[dict]: ...

    @abstractmethod
    async def spotify_aggregate(self, room_token: str, device_id: str, since: date) -> dict: ...

    @abstractmethod
    async def hourly_activity(
        self, room_token: str, device_id: str, tz_offset_min: int, tz_name: str = ""
    ) -> list[int]: ...

    @abstractmethod
    async def room_screen_time(self, room_token: str, tz_offset_min: int, tz_name: str = "") -> list[dict]: ...


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
