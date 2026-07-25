from abc import ABC, abstractmethod
from datetime import date


class ISnapshotReader(ABC):
    @abstractmethod
    async def summary(self, room_token: str, device_id: str, since: date) -> list[dict]: ...

    @abstractmethod
    async def spotify_aggregate(self, room_token: str, device_id: str, since: date) -> dict: ...
