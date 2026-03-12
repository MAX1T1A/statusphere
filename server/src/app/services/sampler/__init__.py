import asyncio
import logging

from app.repositories.snapshot import SnapshotRepository

from .v1.put import put
from .v1.start import start
from .v1.stop import stop


class Sampler:
    def __init__(self, repository: SnapshotRepository, interval: float = 30.0) -> None:
        self._repository = repository
        self._interval = interval
        self._buffer: dict[tuple[str, str], tuple[str, str | None, dict]] = {}
        self._lock = asyncio.Lock()
        self._task: asyncio.Task | None = None

        self._logger = logging.getLogger(__name__)

    put = put
    start = start
    stop = stop

    async def _flush_loop(self) -> None:
        while True:
            await asyncio.sleep(self._interval)
            await self._flush()

    async def _flush(self) -> None:
        async with self._lock:
            if not self._buffer:
                return
            items = list(self._buffer.values())
            self._buffer.clear()

        rows = []
        for room_token, device_name, data in items:
            device_id = data.get("device_id", "")
            rows.append((room_token, device_id, device_name, data))

        try:
            await self._repository.save_batch(rows)
            self._logger.debug("flushed %d snapshots", len(rows))
        except Exception:
            self._logger.exception("flush failed")
