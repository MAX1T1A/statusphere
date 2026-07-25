from app.modules.rooms.application.interfaces import IInviteCodec, IMembershipReader
from app.modules.rooms.domain.exceptions import NoRoomToInvite
from app.shared_kernel.operation import AuthenticatedOperation


class CreateInvite(AuthenticatedOperation):
    pass


class CreateInviteUseCase:
    def __init__(self, reader: IMembershipReader, codec: IInviteCodec) -> None:
        self._reader = reader
        self._codec = codec

    async def execute(self, op: CreateInvite) -> str:
        room_id = await self._reader.owned_room(op.actor.account_id)
        if room_id is None:
            raise NoRoomToInvite()
        return self._codec.sign(room_id)
