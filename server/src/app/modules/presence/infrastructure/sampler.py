import asyncio
import logging

from app.modules.presence.application.interfaces import ISampler, ISnapshotWriter


class Sampler(ISampler):
    def __init__(self, writer: ISnapshotWriter, interval: int) -> None:
        self._writer = writer
        self._interval = interval
        self._buffer: dict[tuple[str, str], tuple[str, dict]] = {}
        self._lock = asyncio.Lock()
        self._task: asyncio.Task | None = None
        self._logger = logging.getLogger(__name__)

    async def put(self, room_token: str, account_id: str, device_id: str, data: dict) -> None:
        async with self._lock:
            self._buffer[(room_token, device_id)] = (account_id, data)

    def start(self) -> None:
        self._task = asyncio.create_task(self._flush_loop())

    async def stop(self) -> None:
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        await self._flush()

    async def _flush_loop(self) -> None:
        while True:
            await asyncio.sleep(self._interval)
            await self._flush()

    async def _flush(self) -> None:
        async with self._lock:
            if not self._buffer:
                return
            pending = dict(self._buffer)
            self._buffer.clear()

        rows = [
            (room_token, account_id, device_id, data)
            for (room_token, device_id), (account_id, data) in pending.items()
        ]

        try:
            await self._writer.save_batch(rows)
            self._logger.debug("flushed %d snapshots", len(rows))
        except Exception:
            self._logger.exception("flush failed; requeueing %d snapshots", len(pending))
            async with self._lock:
                for key, value in pending.items():
                    self._buffer.setdefault(key, value)
