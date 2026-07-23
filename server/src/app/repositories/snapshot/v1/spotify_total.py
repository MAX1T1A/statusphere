import json
from datetime import date

from app.core.crypto import decrypt

QUERY = """
    SELECT data FROM snapshots
    WHERE room_token = $1 AND device_id = $2 AND created_at::date >= $3
"""


async def spotify_total(self, room_token: str, device_id: str, since: date) -> int:
    async with self.pool.acquire() as conn:
        rows = await conn.fetch(QUERY, room_token, device_id, since)

    playing = 0
    for row in rows:
        try:
            payload = json.loads(decrypt(row["data"]))
        except Exception:
            continue
        if payload.get("spotify_status") == "playing":
            playing += 1

    return playing * self.sample_interval
