from datetime import date


async def summary(self, room_token: str, device_id: str, since: date) -> list[dict]:
    query = """
        SELECT
            data->>'active_app' AS app,
            COUNT(*) * $4 AS seconds
        FROM snapshots
        WHERE room_token = $1
            AND device_id = $2
            AND created_at::date >= $3
            AND data->>'active_app' IS NOT NULL
            AND data->>'active_app' != ''
        GROUP BY data->>'active_app'
        ORDER BY seconds DESC
    """

    async with self.pool.acquire() as conn:
        rows = await conn.fetch(query, room_token, device_id, since, self.sample_interval)
        return [dict(r) for r in rows]
