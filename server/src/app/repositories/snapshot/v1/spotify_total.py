from datetime import date


async def spotify_total(self, room_token: str, device_id: str, since: date) -> int:
    query = """
        SELECT COUNT(*) * $4 AS seconds
        FROM snapshots
        WHERE room_token = $1
            AND device_id = $2
            AND created_at::date >= $3
            AND data->>'spotify_status' = 'playing'
    """

    async with self.pool.acquire() as conn:
        row = await conn.fetchrow(query, room_token, device_id, since, self.sample_interval)
        return row["seconds"] if row else 0
