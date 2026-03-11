from __future__ import annotations

import asyncio
from collections.abc import Awaitable, Callable

from app.renderer.tui.widgets import format_cpu, format_memory
from rich.console import Console, Group
from rich.live import Live
from rich.panel import Panel
from rich.table import Table


class TUI:
    def __init__(self, on_ready: Callable[[], Awaitable[None]] | None = None) -> None:
        self._devices: dict[str, dict] = {}
        self._on_ready = on_ready
        self._running = False
        self._console = Console()
        self._live: Live | None = None

    async def run(self) -> None:
        self._running = True
        self._live = Live(
            self._build(),
            console=self._console,
            screen=True,
            refresh_per_second=4,
        )
        self._live.start()

        try:
            if self._on_ready:
                await self._on_ready()

            while self._running:
                await asyncio.sleep(0.1)
        finally:
            self._live.stop()
            self._live = None

    def stop(self) -> None:
        self._running = False

    def update(self, data: dict) -> None:
        device_id = data.get("device_id")
        if not device_id:
            return
        self._devices[device_id] = {k: v for k, v in data.items() if k != "device_id"}
        if self._live:
            self._live.update(self._build())

    def _build(self) -> Group:
        h = self._console.height
        w = self._console.width

        table = Table(expand=True, show_edge=False, pad_edge=True, show_lines=False)
        table.add_column("Device", style="bold cyan", no_wrap=True, ratio=2)
        table.add_column("CPU", justify="right", ratio=1)
        table.add_column("Memory", justify="right", ratio=2)
        table.add_column("Load 1m", justify="right", ratio=1)
        table.add_column("Workspace", style="magenta", ratio=1)
        table.add_column("Window", style="dim", ratio=4, overflow="ellipsis")

        if not self._devices:
            table.add_row("[dim]waiting for devices…[/dim]", "", "", "", "", "")
        else:
            for device_id, snap in sorted(self._devices.items()):
                table.add_row(
                    device_id,
                    format_cpu(snap.get("cpu_percent")),
                    format_memory(snap.get("memory_used_mb"), snap.get("memory_total_mb")),
                    f"{snap['load_avg_1m']:.2f}" if snap.get("load_avg_1m") is not None else "—",
                    snap.get("active_workspace") or "—",
                    snap.get("active_window") or "—",
                )

        panel = Panel(
            table,
            title="[bold]statusphere[/bold]",
            subtitle="[dim]q to quit[/dim]",
            border_style="blue",
            height=h - 1,
        )

        return Group(panel)
