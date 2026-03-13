from datetime import date


async def spotify_daily(self, room_token: str, device_id: str, since: date) -> list[dict]:
    query = """
        SELECT
            created_at::date AS day,
            COUNT(*) * $4 AS seconds
        FROM snapshots
        WHERE room_token = $1
            AND device_id = $2
            AND created_at::date >= $3
            AND data->>'spotify_status' = 'playing'
        GROUP BY created_at::date
        ORDER BY day
    """

    async with self.pool.acquire() as conn:
        rows = await conn.fetch(query, room_token, device_id, since, self.sample_interval)
        return [{"day": str(r["day"]), "seconds": r["seconds"]} for r in rows]
