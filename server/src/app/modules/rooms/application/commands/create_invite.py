from app.modules.rooms.application.interfaces import IInviteCodec, IMembershipReader
from app.modules.rooms.domain.exceptions import NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class CreateInvite(AuthenticatedOperation):
    room: str


class CreateInviteUseCase:
    def __init__(self, reader: IMembershipReader, codec: IInviteCodec) -> None:
        self._reader = reader
        self._codec = codec

    async def execute(self, op: CreateInvite) -> str:
        if not op.room or not await self._reader.is_member(op.room, op.actor.account_id):
            raise NotRoomMember()
        return self._codec.sign(op.room)
