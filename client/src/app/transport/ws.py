from __future__ import annotations

import asyncio
import json
import logging
from collections.abc import Awaitable, Callable

import websockets
from app.collector.snapshot import Snapshot
from app.transport import Transport
from app.utils.device import get_device_id


class WSTransport(Transport):
    def __init__(self, server_url: str, token: str) -> None:
        self._server_url = server_url.rstrip("/")
        self._token = token
        self._device_id = get_device_id()
        self._ws = None

        self.logger = logging.getLogger(__name__)

    async def start(self) -> None:
        url = self._server_url.replace("https://", "wss://").replace("http://", "ws://")
        self._ws = await websockets.connect(
            f"{url}/ws",
            additional_headers={
                "X-Room-Token": self._token,
                "X-Device-Id": self._device_id,
            },
        )
        self.logger.info("ws connected to %s", url)

    async def stop(self) -> None:
        if self._ws:
            await self._ws.close()
            self._ws = None
        self.logger.info("ws disconnected")

    async def send(self, snapshot: Snapshot) -> None:
        if not self._ws:
            raise RuntimeError("Transport is not started")
        data = {k: v for k, v in snapshot.__dict__.items() if v is not None}
        await self._ws.send(json.dumps(data))

    async def listen(self, on_event: Callable[[dict], Awaitable[None]]) -> None:
        if not self._ws:
            raise RuntimeError("Transport is not started")
        try:
            async for raw in self._ws:
                data = json.loads(raw)
                await on_event(data)
        except websockets.ConnectionClosed:
            self.logger.info("ws connection closed")
