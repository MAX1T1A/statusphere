import asyncpg
from app.core.config import get_settings
from asyncpg.pool import Pool


async def provide_pool() -> Pool:
    postgres = get_settings().postgres
    pool = await asyncpg.create_pool(
        user=postgres.username,
        password=postgres.password,
        database=postgres.dbname,
        host=postgres.host,
        port=postgres.port,
        max_size=postgres.pool_size,
    )

    async with pool.acquire() as conn:
        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS snapshots (
                id BIGSERIAL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                room_token TEXT NOT NULL,
                account_id TEXT NOT NULL,
                device_id TEXT NOT NULL,
                data BYTEA NOT NULL
            )
        """
        )

        await conn.execute("ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS account_id TEXT NOT NULL DEFAULT ''")

        await conn.execute(
            """
            CREATE INDEX IF NOT EXISTS idx_snapshots_lookup
            ON snapshots (room_token, device_id, created_at DESC)
        """
        )

        try:
            await conn.execute("SELECT create_hypertable('snapshots', 'created_at', if_not_exists => TRUE)")
        except asyncpg.UndefinedFunctionError:
            pass

        try:
            await conn.execute("SELECT add_retention_policy('snapshots', INTERVAL '7 days', if_not_exists => TRUE)")
        except asyncpg.UndefinedFunctionError:
            pass

        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS accounts (
                account_id TEXT PRIMARY KEY,
                secret_verifier TEXT NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now()
            )
        """
        )

        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS devices (
                device_id TEXT PRIMARY KEY,
                account_id TEXT NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
                name TEXT,
                revoked BOOLEAN NOT NULL DEFAULT FALSE,
                linked_at TIMESTAMPTZ NOT NULL DEFAULT now()
            )
        """
        )

        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS room_members (
                room_id TEXT NOT NULL,
                account_id TEXT NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
                role TEXT NOT NULL DEFAULT 'member',
                joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                PRIMARY KEY (room_id, account_id)
            )
        """
        )

        await conn.execute("CREATE INDEX IF NOT EXISTS idx_devices_account ON devices (account_id)")
        await conn.execute("CREATE INDEX IF NOT EXISTS idx_room_members_account ON room_members (account_id)")

    return pool
