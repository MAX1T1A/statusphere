import asyncio


async def publish(self, token: str, device_id: str, data: dict) -> None:
    room = self._storage.get_or_create(token)
    for subscriber in room.subscribers:
        if subscriber.device_id != device_id:
            try:
                subscriber.queue.put_nowait({"device_id": device_id, **data})
            except asyncio.QueueFull:
                pass
