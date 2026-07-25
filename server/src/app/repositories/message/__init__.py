from datetime import datetime

from app.platform.crypto import decrypt, encrypt
from asyncpg.pool import Pool

GROUP_LIMIT = 300
DM_LIMIT = 100
MAX_TEXT = 500

_DM_PAIR = "((from_account = $2 AND to_account = $3) OR (from_account = $3 AND to_account = $2))"


class MessageRepository:
    def __init__(self, pool: Pool) -> None:
        self.pool = pool

    async def save(self, room_token: str, from_account: str, to_account: str, text: str) -> datetime:
        blob = encrypt(text[:MAX_TEXT].encode())
        async with self.pool.acquire() as conn:
            created = await conn.fetchval(
                "INSERT INTO messages (room_token, from_account, to_account, data) "
                "VALUES ($1, $2, $3, $4) RETURNING created_at",
                room_token,
                from_account,
                to_account,
                blob,
            )
            if to_account == "":
                await conn.execute(
                    "DELETE FROM messages WHERE room_token = $1 AND to_account = '' AND id NOT IN "
                    "(SELECT id FROM messages WHERE room_token = $1 AND to_account = '' "
                    "ORDER BY created_at DESC LIMIT $2)",
                    room_token,
                    GROUP_LIMIT,
                )
            else:
                await conn.execute(
                    f"DELETE FROM messages WHERE room_token = $1 AND to_account <> '' AND {_DM_PAIR} "
                    f"AND id NOT IN (SELECT id FROM messages WHERE room_token = $1 AND to_account <> '' AND {_DM_PAIR} "
                    "ORDER BY created_at DESC LIMIT $4)",
                    room_token,
                    from_account,
                    to_account,
                    DM_LIMIT,
                )
        return created

    async def history(self, room_token: str, account: str) -> list[dict]:
        async with self.pool.acquire() as conn:
            group_rows = await conn.fetch(
                "SELECT from_account, to_account, data, created_at FROM messages "
                "WHERE room_token = $1 AND to_account = '' ORDER BY created_at DESC LIMIT $2",
                room_token,
                GROUP_LIMIT,
            )
            dm_rows = await conn.fetch(
                "SELECT from_account, to_account, data, created_at FROM ("
                "  SELECT from_account, to_account, data, created_at, ROW_NUMBER() OVER ("
                "    PARTITION BY (CASE WHEN from_account = $2 THEN to_account ELSE from_account END)"
                "    ORDER BY created_at DESC) AS rn"
                "  FROM messages WHERE room_token = $1 AND to_account <> '' AND (from_account = $2 OR to_account = $2)"
                ") t WHERE rn <= $3",
                room_token,
                account,
                DM_LIMIT,
            )

        out: list[dict] = []
        for row in list(group_rows) + list(dm_rows):
            try:
                text = decrypt(row["data"]).decode()
            except Exception:
                continue
            out.append(
                {
                    "from": row["from_account"],
                    "to": row["to_account"],
                    "text": text,
                    "at": row["created_at"].isoformat(),
                }
            )
        out.sort(key=lambda m: m["at"])
        return out
