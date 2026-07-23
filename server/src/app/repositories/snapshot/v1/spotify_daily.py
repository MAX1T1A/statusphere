import json
from collections import defaultdict
from datetime import date

from app.core.crypto import decrypt

QUERY = """
    SELECT created_at::date AS day, data FROM snapshots
    WHERE room_token = $1 AND device_id = $2 AND created_at::date >= $3
"""


async def spotify_daily(self, room_token: str, device_id: str, since: date) -> list[dict]:
    async with self.pool.acquire() as conn:
        rows = await conn.fetch(QUERY, room_token, device_id, since)

    per_day: dict = defaultdict(int)
    for row in rows:
        try:
            payload = json.loads(decrypt(row["data"]))
        except Exception:
            continue
        if payload.get("spotify_status") == "playing":
            per_day[row["day"]] += 1

    return [{"day": str(day), "seconds": count * self.sample_interval} for day, count in sorted(per_day.items())]
