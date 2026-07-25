from app.modules.rooms.application.interfaces import IMembershipReader
from asyncpg.pool import Pool


class MembershipReader(IMembershipReader):
    def __init__(self, pool: Pool) -> None:
        self._pool = pool

    async def is_member(self, room_id: str, account_id: str) -> bool:
        async with self._pool.acquire() as conn:
            found = await conn.fetchval(
                "SELECT TRUE FROM room_members WHERE room_id = $1 AND account_id = $2",
                room_id,
                account_id,
            )
            return found is True

    async def owned_room(self, account_id: str) -> str | None:
        async with self._pool.acquire() as conn:
            return await conn.fetchval(
                "SELECT room_id FROM room_members WHERE account_id = $1 AND role = 'owner' ORDER BY joined_at LIMIT 1",
                account_id,
            )

    async def list_members(self, room_id: str) -> list[dict]:
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT account_id, role, joined_at FROM room_members WHERE room_id = $1 ORDER BY joined_at",
                room_id,
            )
            return [dict(row) for row in rows]

    async def role_of(self, room_id: str, account_id: str) -> str | None:
        async with self._pool.acquire() as conn:
            return await conn.fetchval(
                "SELECT role FROM room_members WHERE room_id = $1 AND account_id = $2",
                room_id,
                account_id,
            )
