import asyncio


def start(self) -> None:
    self._task = asyncio.create_task(self._flush_loop())
