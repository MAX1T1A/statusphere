import asyncio


async def publish(self, token: str, account_id: str, device_id: str, data: dict) -> None:
    room = self._storage.get_or_create(token)
    for subscriber in room.subscribers:
        if subscriber.device_id != device_id:
            try:
                subscriber.queue.put_nowait({**data, "account_id": account_id, "device_id": device_id})
            except asyncio.QueueFull:
                pass
