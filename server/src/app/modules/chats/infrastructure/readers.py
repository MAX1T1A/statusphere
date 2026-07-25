from asyncpg.pool import Pool

from app.modules.chats.application.dto import MessageDTO
from app.modules.chats.application.interfaces import IMessageReader
from app.modules.chats.domain.message import DM_LIMIT, GROUP_LIMIT
from app.platform.crypto import decrypt

_GROUP_QUERY = (
    "SELECT from_account, to_account, data, created_at FROM messages "
    "WHERE room_token = $1 AND to_account = '' ORDER BY created_at DESC LIMIT $2"
)

_DM_QUERY = (
    "SELECT from_account, to_account, data, created_at FROM ("
    "  SELECT from_account, to_account, data, created_at, ROW_NUMBER() OVER ("
    "    PARTITION BY (CASE WHEN from_account = $2 THEN to_account ELSE from_account END)"
    "    ORDER BY created_at DESC) AS rn"
    "  FROM messages WHERE room_token = $1 AND to_account <> '' AND (from_account = $2 OR to_account = $2)"
    ") t WHERE rn <= $3"
)


class MessageReader(IMessageReader):
    def __init__(self, pool: Pool) -> None:
        self._pool = pool

    async def history(self, room_token: str, account: str) -> list[MessageDTO]:
        async with self._pool.acquire() as conn:
            group_rows = await conn.fetch(_GROUP_QUERY, room_token, GROUP_LIMIT)
            dm_rows = await conn.fetch(_DM_QUERY, room_token, account, DM_LIMIT)

        out: list[MessageDTO] = []
        for row in list(group_rows) + list(dm_rows):
            try:
                text = decrypt(row["data"]).decode()
            except Exception:
                continue
            out.append(
                MessageDTO(
                    from_account=row["from_account"],
                    to_account=row["to_account"],
                    text=text,
                    at=row["created_at"].isoformat(),
                )
            )
        out.sort(key=lambda m: m.at)
        return out
