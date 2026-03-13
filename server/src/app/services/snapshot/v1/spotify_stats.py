from datetime import date


async def spotify_stats(self, room_token: str, device_id: str, period: str, since: date) -> dict:
    total = await self._repository.spotify_total(room_token, device_id, since)
    daily = await self._repository.spotify_daily(room_token, device_id, since)

    return {
        "device_id": device_id,
        "period": period,
        "since": str(since),
        "total_seconds": total,
        "daily": daily,
    }
