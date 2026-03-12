from datetime import date


async def summary(self, room_token: str, device_id: str, period: str, since: date) -> list[dict]:
    rows = await self._repository.summary(room_token, device_id, since)

    return {
        "device_id": device_id,
        "period": period,
        "since": str(since),
        "apps": [{"app": r["app"], "seconds": r["seconds"]} for r in rows],
    }
