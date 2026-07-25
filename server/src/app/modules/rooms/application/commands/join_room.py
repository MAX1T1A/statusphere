from typing import Callable

from app.modules.rooms.application.interfaces import IInviteCodec, IRoomsUnitOfWork
from app.modules.rooms.domain.exceptions import InvalidOrExpiredInvite
from app.shared_kernel.operation import AuthenticatedOperation


class JoinRoom(AuthenticatedOperation):
    code: str


class JoinRoomUseCase:
    def __init__(self, uow_factory: Callable[[], IRoomsUnitOfWork], codec: IInviteCodec) -> None:
        self._uow_factory = uow_factory
        self._codec = codec

    async def execute(self, op: JoinRoom) -> str:
        room_id = self._codec.verify(op.code)
        if room_id is None:
            raise InvalidOrExpiredInvite()
        async with self._uow_factory() as uow:
            await uow.memberships.add_member(room_id, op.actor.account_id, "member")
        return room_id
