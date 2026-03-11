from collections.abc import Awaitable, Callable

from app.renderer.tui import TUI


class FeedRenderer:
    def __init__(self, on_ready: Callable[[], Awaitable[None]] | None = None) -> None:
        self._tui = TUI(on_ready=on_ready)

    async def run(self) -> None:
        await self._tui.run()

    def stop(self) -> None:
        self._tui.stop()

    async def update(self, data: dict) -> None:
        self._tui.update(data)
