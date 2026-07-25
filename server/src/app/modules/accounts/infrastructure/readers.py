from asyncpg.pool import Pool

from app.modules.accounts.application.interfaces import IAccountReader


class AccountReader(IAccountReader):
    def __init__(self, pool: Pool) -> None:
        self._pool = pool

    async def get_verifier(self, account_id: str) -> str | None:
        async with self._pool.acquire() as conn:
            return await conn.fetchval("SELECT secret_verifier FROM accounts WHERE account_id = $1", account_id)

    async def is_device_active(self, account_id: str, device_id: str) -> bool:
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT revoked FROM devices WHERE account_id = $1 AND device_id = $2",
                account_id,
                device_id,
            )
            return row is not None and not row["revoked"]

    async def name_of(self, account_id: str) -> str:
        async with self._pool.acquire() as conn:
            return await conn.fetchval("SELECT name FROM accounts WHERE account_id = $1", account_id) or ""

    async def list_devices(self, account_id: str) -> list[dict]:
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT device_id, name, revoked, linked_at FROM devices WHERE account_id = $1 ORDER BY linked_at",
                account_id,
            )
            return [dict(row) for row in rows]
