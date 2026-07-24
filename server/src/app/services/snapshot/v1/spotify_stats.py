from datetime import date


async def spotify_stats(self, room_token: str, device_id: str, period: str, since: date) -> dict:
    aggregate = await self._repository.spotify_aggregate(room_token, device_id, since)

    return {
        "device_id": device_id,
        "period": period,
        "since": str(since),
        **aggregate,
    }
