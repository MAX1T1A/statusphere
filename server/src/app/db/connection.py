import asyncpg
from app.core.config import postgres_config
from asyncpg.pool import Pool


async def provide_pool() -> Pool:
    pool = await asyncpg.create_pool(
        user=postgres_config.username,
        password=postgres_config.password,
        database=postgres_config.dbname,
        host=postgres_config.host,
        port=postgres_config.port,
        max_size=postgres_config.pool_size,
    )

    async with pool.acquire() as conn:
        await conn.execute(
            """
            CREATE TABLE IF NOT EXISTS snapshots (
                id BIGSERIAL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                room_token TEXT NOT NULL,
                device_id TEXT NOT NULL,
                device_name TEXT,
                data JSONB NOT NULL
            )
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
