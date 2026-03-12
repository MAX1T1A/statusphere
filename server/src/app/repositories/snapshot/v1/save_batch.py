import json


async def save_batch(self, rows: list[tuple[str, str, str | None, dict]]) -> None:
    async with self.pool.acquire() as conn:
        await conn.executemany(
            "INSERT INTO snapshots (room_token, device_id, device_name, data) VALUES ($1, $2, $3, $4)",
            [(token, did, name, json.dumps(data)) for token, did, name, data in rows],
        )
