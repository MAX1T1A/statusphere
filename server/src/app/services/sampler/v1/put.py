async def put(self, room_token: str, account_id: str, device_id: str, data: dict) -> None:
    async with self._lock:
        self._buffer[(room_token, device_id)] = (account_id, data)
