from abc import ABC, abstractmethod
from typing import Awaitable, Callable

from app.collector.snapshot import Snapshot


class Transport(ABC):
    @abstractmethod
    async def start(self) -> None: ...

    @abstractmethod
    async def stop(self) -> None: ...

    @abstractmethod
    async def send(self, snapshot: Snapshot) -> None: ...

    @abstractmethod
    async def listen(self, on_event: Callable[[dict], Awaitable[None]]) -> None: ...
