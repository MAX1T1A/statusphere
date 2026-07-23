import json

from app.core.crypto import encrypt


async def save_batch(self, rows: list[tuple[str, str, dict]]) -> None:
    async with self.pool.acquire() as conn:
        await conn.executemany(
            "INSERT INTO snapshots (room_token, device_id, data) VALUES ($1, $2, $3)",
            [(token, did, encrypt(json.dumps(data).encode())) for token, did, data in rows],
        )
