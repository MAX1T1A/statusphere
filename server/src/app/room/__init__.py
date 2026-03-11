import asyncio
from collections.abc import AsyncGenerator

from .storage import RoomStorage, Subscriber


class RoomManager:
    def __init__(self) -> None:
        self._storage = RoomStorage()

    async def publish(self, token: str, device_id: str, data: dict) -> None:
        room = self._storage.get_or_create(token)
        for subscriber in room.subscribers:
            if subscriber.device_id != device_id:
                await subscriber.queue.put({"device_id": device_id, **data})

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
