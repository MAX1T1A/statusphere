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
                device_id TEXT NOT NULL,
                data BYTEA NOT NULL
            )
        """
        )

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

    return pool
