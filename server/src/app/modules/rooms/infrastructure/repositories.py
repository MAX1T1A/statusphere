import asyncpg
from app.modules.rooms.application.interfaces import IMembershipRepository


class MembershipRepository(IMembershipRepository):
    def __init__(self, conn: asyncpg.Connection) -> None:
        self._conn = conn

    async def add_member(self, room_id: str, account_id: str, role: str = "member") -> None:
        await self._conn.execute(
            "INSERT INTO room_members (room_id, account_id, role) VALUES ($1, $2, $3) "
            "ON CONFLICT (room_id, account_id) DO NOTHING",
            room_id,
            account_id,
            role,
        )

    async def remove_member(self, room_id: str, account_id: str) -> None:
        await self._conn.execute(
            "DELETE FROM room_members WHERE room_id = $1 AND account_id = $2",
            room_id,
            account_id,
        )
