from typing import Callable

from app.modules.rooms.application.interfaces import IMembershipReader, IRoomsUnitOfWork
from app.modules.rooms.domain.policy import can_manage
from app.shared_kernel.operation import AuthenticatedOperation


class KickMember(AuthenticatedOperation):
    room: str
    target_account_id: str


class KickMemberUseCase:
    def __init__(self, reader: IMembershipReader, uow_factory: Callable[[], IRoomsUnitOfWork]) -> None:
        self._reader = reader
        self._uow_factory = uow_factory

    async def execute(self, op: KickMember) -> bool:
        if op.actor.account_id == op.target_account_id:
            return False
        actor_role = await self._reader.role_of(op.room, op.actor.account_id)
        target_role = await self._reader.role_of(op.room, op.target_account_id)
        if not can_manage(actor_role, target_role):
            return False
        async with self._uow_factory() as uow:
            await uow.memberships.remove_member(op.room, op.target_account_id)
        return True
