from typing import Callable

from app.modules.rooms.application.interfaces import IMembershipReader, IRoomsUnitOfWork
from app.platform.security import generate_room_id


class RoomDirectory:
    def __init__(self, uow_factory: Callable[[], IRoomsUnitOfWork], reader: IMembershipReader) -> None:
        self._uow_factory = uow_factory
        self._reader = reader

    async def create_room_for_owner(self, account_id: str) -> str:
        room_id = generate_room_id()
        async with self._uow_factory() as uow:
            await uow.memberships.add_member(room_id, account_id, "owner")
        return room_id

    async def owned_room(self, account_id: str) -> str | None:
        return await self._reader.owned_room(account_id)
