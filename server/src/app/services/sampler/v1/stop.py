import asyncio


async def stop(self) -> None:
    if self._task:
        self._task.cancel()
        try:
            await self._task
        except asyncio.CancelledError:
            pass
    await self._flush()
