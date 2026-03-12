import asyncio
from typing import AsyncGenerator

from app.services.room.storage import Subscriber


async def subscribe(self, token: str, device_id: str) -> AsyncGenerator[dict, None]:
    queue: asyncio.Queue = asyncio.Queue()
    room = self._storage.get_or_create(token)
    room.subscribers.append(Subscriber(device_id=device_id, queue=queue))
    try:
        while True:
            data = await queue.get()
            yield data
    finally:
        self._storage.remove_subscriber(token, device_id)
