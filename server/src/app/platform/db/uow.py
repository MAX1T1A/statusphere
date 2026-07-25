from types import TracebackType
from typing import Self

import asyncpg
from asyncpg.pool import Pool


class BaseUnitOfWork:
    def __init__(self, pool: Pool) -> None:
        self._pool = pool
        self._conn: asyncpg.Connection | None = None
        self._tx: asyncpg.transaction.Transaction | None = None

    def _bind(self, conn: asyncpg.Connection) -> None:
        raise NotImplementedError

    async def __aenter__(self) -> Self:
        self._conn = await self._pool.acquire()
        self._tx = self._conn.transaction()
        await self._tx.start()
        self._bind(self._conn)
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None:
        try:
            if exc_type is not None:
                await self._tx.rollback()
            else:
                await self._tx.commit()
        finally:
            await self._pool.release(self._conn)
            self._conn = None
            self._tx = None
