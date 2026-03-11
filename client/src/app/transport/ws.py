from __future__ import annotations

import asyncio
import json
import logging
from collections.abc import Awaitable, Callable

import websockets
from app.collector.snapshot import Snapshot
from app.transport import Transport
from app.utils.device import get_device_id

RECONNECT_DELAY = 3
PING_INTERVAL = 20
PING_TIMEOUT = 10


class WSTransport(Transport):
    def __init__(self, server_url: str, token: str) -> None:
        self._url = server_url.rstrip("/").replace("https://", "wss://").replace("http://", "ws://") + "/ws"
        self._token = token
        self._device_id = get_device_id()
        self._ws = None
        self._running = False

        self.logger = logging.getLogger(__name__)

    async def start(self) -> None:
        self._running = True
        await self._connect()

    async def stop(self) -> None:
        self._running = False
        if self._ws:
            await self._ws.close()
            self._ws = None

    async def send(self, snapshot: Snapshot) -> None:
        if not self._ws:
            return
        data = {k: v for k, v in snapshot.__dict__.items() if v is not None}
        try:
            await self._ws.send(json.dumps(data))
        except websockets.ConnectionClosed:
            self.logger.warning("send failed, connection lost")

    async def listen(self, on_event: Callable[[dict], Awaitable[None]]) -> None:
        while self._running:
            if not self._ws:
                await self._reconnect()
                continue
            try:
                async for raw in self._ws:
                    data = json.loads(raw)
                    await on_event(data)
            except websockets.ConnectionClosed:
                self.logger.warning("connection lost")
            except Exception:
                self.logger.exception("listen error")

            if self._running:
                await self._reconnect()

    async def _connect(self) -> None:
        self._ws = await websockets.connect(
            self._url,
            additional_headers={
                "X-Room-Token": self._token,
                "X-Device-Id": self._device_id,
            },
            ping_interval=PING_INTERVAL,
            ping_timeout=PING_TIMEOUT,
        )
        self.logger.info("ws connected")

    async def _reconnect(self) -> None:
        self._ws = None
        while self._running:
            try:
                self.logger.info("reconnecting in %ds...", RECONNECT_DELAY)
                await asyncio.sleep(RECONNECT_DELAY)
                await self._connect()
                return
            except Exception:
                self.logger.warning("reconnect failed, retrying...")
