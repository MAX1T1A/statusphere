import json

from app.core.crypto import encrypt


async def save_batch(self, rows: list[tuple[str, str, str, dict]]) -> None:
    async with self.pool.acquire() as conn:
        await conn.executemany(
            "INSERT INTO snapshots (room_token, account_id, device_id, data) VALUES ($1, $2, $3, $4)",
            [
                (room_token, account_id, device_id, encrypt(json.dumps(data).encode()))
                for room_token, account_id, device_id, data in rows
            ],
        )
