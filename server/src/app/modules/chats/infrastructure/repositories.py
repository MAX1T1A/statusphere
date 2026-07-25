from datetime import datetime

import asyncpg

from app.modules.chats.application.interfaces import IMessageRepository
from app.modules.chats.domain.message import DM_LIMIT, GROUP_LIMIT
from app.platform.crypto import encrypt

_DM_PAIR = "((from_account = $2 AND to_account = $3) OR (from_account = $3 AND to_account = $2))"


class MessageRepository(IMessageRepository):
    def __init__(self, conn: asyncpg.Connection) -> None:
        self._conn = conn

    async def add(self, room_token: str, from_account: str, to_account: str, text: str) -> datetime:
        blob = encrypt(text.encode())
        created = await self._conn.fetchval(
            "INSERT INTO messages (room_token, from_account, to_account, data) "
            "VALUES ($1, $2, $3, $4) RETURNING created_at",
            room_token,
            from_account,
            to_account,
            blob,
        )
        if to_account == "":
            await self._conn.execute(
                "DELETE FROM messages WHERE room_token = $1 AND to_account = '' AND id NOT IN "
                "(SELECT id FROM messages WHERE room_token = $1 AND to_account = '' "
                "ORDER BY created_at DESC LIMIT $2)",
                room_token,
                GROUP_LIMIT,
            )
        else:
            await self._conn.execute(
                f"DELETE FROM messages WHERE room_token = $1 AND to_account <> '' AND {_DM_PAIR} "
                f"AND id NOT IN (SELECT id FROM messages WHERE room_token = $1 AND to_account <> '' AND {_DM_PAIR} "
                "ORDER BY created_at DESC LIMIT $4)",
                room_token,
                from_account,
                to_account,
                DM_LIMIT,
            )
        return created
