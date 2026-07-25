import json
from collections import Counter
from datetime import date

from app.platform.crypto import decrypt

QUERY = """
    SELECT data FROM snapshots
    WHERE room_token = $1 AND device_id = $2 AND created_at::date >= $3
"""


async def summary(self, room_token: str, device_id: str, since: date) -> list[dict]:
    async with self.pool.acquire() as conn:
        rows = await conn.fetch(QUERY, room_token, device_id, since)

    counts: Counter = Counter()
    for row in rows:
        try:
            payload = json.loads(decrypt(row["data"]))
        except Exception:
            continue
        app = payload.get("active_app")
        if app:
            counts[app] += 1

    return [{"app": app, "seconds": n * self.sample_interval} for app, n in counts.most_common()]
