from typing import Callable

from app.modules.rooms.application.interfaces import IMembershipReader, IRoomsUnitOfWork
from app.modules.rooms.domain.policy import can_kick
from app.shared_kernel.operation import AuthenticatedOperation


class KickMember(AuthenticatedOperation):
    target_account_id: str


class KickMemberUseCase:
    def __init__(self, reader: IMembershipReader, uow_factory: Callable[[], IRoomsUnitOfWork]) -> None:
        self._reader = reader
        self._uow_factory = uow_factory

    async def execute(self, op: KickMember) -> bool:
        room_id = await self._reader.owned_room(op.actor.account_id)
        if room_id is None:
            return False
        role = await self._reader.role_of(room_id, op.target_account_id)
        if not can_kick(op.actor.account_id, op.target_account_id, role):
            return False
        async with self._uow_factory() as uow:
            await uow.memberships.remove_member(room_id, op.target_account_id)
        return True
