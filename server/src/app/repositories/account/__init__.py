from asyncpg.pool import Pool


class AccountRepository:
    def __init__(self, pool: Pool) -> None:
        self.pool = pool

    async def create_account(self, account_id: str, secret_verifier: str) -> None:
        async with self.pool.acquire() as conn:
            await conn.execute(
                "INSERT INTO accounts (account_id, secret_verifier) VALUES ($1, $2)",
                account_id,
                secret_verifier,
            )

    async def get_verifier(self, account_id: str) -> str | None:
        async with self.pool.acquire() as conn:
            return await conn.fetchval("SELECT secret_verifier FROM accounts WHERE account_id = $1", account_id)

    async def create_device(self, device_id: str, account_id: str, name: str | None) -> None:
        async with self.pool.acquire() as conn:
            await conn.execute(
                "INSERT INTO devices (device_id, account_id, name) VALUES ($1, $2, $3)",
                device_id,
                account_id,
                name,
            )

    async def list_devices(self, account_id: str) -> list[dict]:
        async with self.pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT device_id, name, revoked, linked_at FROM devices WHERE account_id = $1 ORDER BY linked_at",
                account_id,
            )
            return [dict(row) for row in rows]

    async def revoke_device(self, account_id: str, device_id: str) -> bool:
        async with self.pool.acquire() as conn:
            status = await conn.execute(
                "UPDATE devices SET revoked = TRUE WHERE account_id = $1 AND device_id = $2",
                account_id,
                device_id,
            )
            return status.split()[-1] != "0"

    async def is_device_active(self, account_id: str, device_id: str) -> bool:
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT revoked FROM devices WHERE account_id = $1 AND device_id = $2",
                account_id,
                device_id,
            )
            return row is not None and not row["revoked"]
