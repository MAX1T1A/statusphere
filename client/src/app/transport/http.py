import json
from typing import Awaitable, Callable

import httpx
from app.collector.snapshot import Snapshot
from app.transport import Transport
from app.utils.device import get_device_id


class HTTPTransport(Transport):
    def __init__(self, server_url: str, token: str) -> None:
        self._server_url = server_url.rstrip("/")
        self._token = token
        self._device_id = get_device_id()
        self._client: httpx.AsyncClient | None = None

    async def start(self) -> None:
        self._client = httpx.AsyncClient()

    async def stop(self) -> None:
        if self._client:
            await self._client.aclose()
            self._client = None

    async def send(self, snapshot: Snapshot) -> None:
        if not self._client:
            raise RuntimeError("Transport is not started")
        await self._client.post(
            f"{self._server_url}/status",
            json=self._serialize(snapshot),
            headers={
                "X-Room-Token": self._token,
                "X-Device-Id": self._device_id,
            },
        )

    async def listen(self, on_event: Callable[[dict], Awaitable[None]]) -> None:
        if not self._client:
            raise RuntimeError("Transport is not started")
        async with self._client.stream(
            "GET",
            f"{self._server_url}/feed",
            headers={
                "X-Room-Token": self._token,
                "X-Device-Id": self._device_id,
            },
            timeout=httpx.Timeout(timeout=None),
        ) as response:
            async for line in response.aiter_lines():
                if line.startswith("data: "):
                    data = json.loads(line.removeprefix("data: "))
                    await on_event(data)

    def _serialize(self, snapshot: Snapshot) -> dict:
        return {k: v for k, v in snapshot.__dict__.items() if v is not None}
