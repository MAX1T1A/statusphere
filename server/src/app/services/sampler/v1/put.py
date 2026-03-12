async def put(self, room_token: str, device_id: str, data: dict) -> None:
    async with self._lock:
        self._buffer[(room_token, device_id)] = (
            room_token,
            data.get("device_name"),
            data,
        )
