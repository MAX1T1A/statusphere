import asyncpg

from app.modules.accounts.application.interfaces import IAccountRepository


class AccountRepository(IAccountRepository):
    def __init__(self, conn: asyncpg.Connection) -> None:
        self._conn = conn

    async def create_account(self, account_id: str, secret_verifier: str) -> None:
        await self._conn.execute(
            "INSERT INTO accounts (account_id, secret_verifier) VALUES ($1, $2)",
            account_id,
            secret_verifier,
        )

    async def create_device(self, device_id: str, account_id: str, name: str | None) -> None:
        await self._conn.execute(
            "INSERT INTO devices (device_id, account_id, name) VALUES ($1, $2, $3)",
            device_id,
            account_id,
            name,
        )

    async def set_name(self, account_id: str, name: str) -> None:
        await self._conn.execute("UPDATE accounts SET name = $2 WHERE account_id = $1", account_id, name)

    async def revoke_device(self, account_id: str, device_id: str) -> bool:
        status = await self._conn.execute(
            "UPDATE devices SET revoked = TRUE WHERE account_id = $1 AND device_id = $2",
            account_id,
            device_id,
        )
        return status.split()[-1] != "0"
