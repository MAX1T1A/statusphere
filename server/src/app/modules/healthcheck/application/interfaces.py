from abc import ABC, abstractmethod


class IHealthProbe(ABC):
    @abstractmethod
    async def database(self) -> dict: ...

    @abstractmethod
    async def snapshots(self) -> dict: ...
