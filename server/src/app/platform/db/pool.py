import asyncpg
from asyncpg.pool import Pool

from app.platform.config import get_settings
from app.platform.db.schema import create_all_tables


async def provide_pool() -> Pool:
    postgres = get_settings().postgres
    pool = await asyncpg.create_pool(
        user=postgres.username,
        password=postgres.password,
        database=postgres.dbname,
        host=postgres.host,
        port=postgres.port,
        max_size=postgres.pool_size,
        server_settings={"timezone": "UTC"},
    )

    async with pool.acquire() as conn:
        await create_all_tables(conn)

    return pool
