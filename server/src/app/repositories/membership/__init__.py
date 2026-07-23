from asyncpg.pool import Pool


class MembershipRepository:
    def __init__(self, pool: Pool) -> None:
        self.pool = pool

    async def add_member(self, room_id: str, account_id: str, role: str = "member") -> None:
        async with self.pool.acquire() as conn:
            await conn.execute(
                "INSERT INTO room_members (room_id, account_id, role) VALUES ($1, $2, $3) "
                "ON CONFLICT (room_id, account_id) DO NOTHING",
                room_id,
                account_id,
                role,
            )

    async def remove_member(self, room_id: str, account_id: str) -> None:
        async with self.pool.acquire() as conn:
            await conn.execute(
                "DELETE FROM room_members WHERE room_id = $1 AND account_id = $2",
                room_id,
                account_id,
            )

    async def is_member(self, room_id: str, account_id: str) -> bool:
        async with self.pool.acquire() as conn:
            found = await conn.fetchval(
                "SELECT TRUE FROM room_members WHERE room_id = $1 AND account_id = $2",
                room_id,
                account_id,
            )
            return found is True

    async def list_members(self, room_id: str) -> list[dict]:
        async with self.pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT account_id, role, joined_at FROM room_members WHERE room_id = $1 ORDER BY joined_at",
                room_id,
            )
            return [dict(row) for row in rows]

    async def role_of(self, room_id: str, account_id: str) -> str | None:
        async with self.pool.acquire() as conn:
            return await conn.fetchval(
                "SELECT role FROM room_members WHERE room_id = $1 AND account_id = $2",
                room_id,
                account_id,
            )

    async def owned_room(self, account_id: str) -> str | None:
        async with self.pool.acquire() as conn:
            return await conn.fetchval(
                "SELECT room_id FROM room_members WHERE account_id = $1 AND role = 'owner' ORDER BY joined_at LIMIT 1",
                account_id,
            )
