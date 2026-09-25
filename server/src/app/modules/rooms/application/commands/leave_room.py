from typing import Callable

from app.modules.rooms.application.interfaces import IMembershipReader, IRoomsUnitOfWork
from app.modules.rooms.domain.policy import can_leave
from app.shared_kernel.operation import AuthenticatedOperation


class LeaveRoom(AuthenticatedOperation):
    room: str


class LeaveRoomUseCase:
    def __init__(self, reader: IMembershipReader, uow_factory: Callable[[], IRoomsUnitOfWork]) -> None:
        self._reader = reader
        self._uow_factory = uow_factory

    async def execute(self, op: LeaveRoom) -> bool:
        role = await self._reader.role_of(op.room, op.actor.account_id)
        if not can_leave(role):
            return False
        async with self._uow_factory() as uow:
            await uow.memberships.remove_member(op.room, op.actor.account_id)
        return True
