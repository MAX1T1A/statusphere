from datetime import datetime

from app.repositories.message import MessageRepository


class MessageService:
    def __init__(self, repository: MessageRepository) -> None:
        self._repository = repository

    async def save(self, room_token: str, from_account: str, to_account: str, text: str) -> datetime:
        return await self._repository.save(room_token, from_account, to_account, text)

    async def history(self, room_token: str, account: str) -> list[dict]:
        return await self._repository.history(room_token, account)
