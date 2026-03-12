async def publish(self, token: str, device_id: str, data: dict) -> None:
    room = self._storage.get_or_create(token)
    for subscriber in room.subscribers:
        if subscriber.device_id != device_id:
            await subscriber.queue.put({"device_id": device_id, **data})
