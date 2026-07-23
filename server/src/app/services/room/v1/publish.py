import asyncio


async def publish(self, token: str, account_id: str, account_name: str, device_id: str, data: dict) -> None:
    room = self._storage.get_or_create(token)
    stamped = {**data, "account_id": account_id, "account_name": account_name, "device_id": device_id}
    for subscriber in room.subscribers:
        if subscriber.device_id != device_id:
            try:
                subscriber.queue.put_nowait(stamped)
            except asyncio.QueueFull:
                pass
