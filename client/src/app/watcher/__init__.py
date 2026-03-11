import asyncio
from collections.abc import Awaitable, Callable

from app.collector import SystemCollector
from app.collector.snapshot import Snapshot
from app.detector import SystemDetector

from .diff import has_diff


class SystemWatcher:
    def __init__(self, on_change: Callable[[Snapshot], Awaitable[None]], interval: float = 2.0) -> None:
        context = SystemDetector().detect()
        self._collector = SystemCollector(context)
        self._on_change = on_change
        self._interval = interval
        self._task: asyncio.Task | None = None
        self._last_snapshot: Snapshot | None = None

    async def start(self) -> None:
        self._task = asyncio.create_task(self._watch())

    async def stop(self) -> None:
        if self._task:
            self._task.cancel()
            self._task = None

    async def _watch(self) -> None:
        while True:
            snapshot = self._collector.collect()

            if self._last_snapshot is None or has_diff(self._last_snapshot, snapshot):
                self._last_snapshot = snapshot
                await self._on_change(snapshot)

            await asyncio.sleep(self._interval)
