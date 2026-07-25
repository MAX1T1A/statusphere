from app.modules.chats.application.dto import MessageDTO
from app.modules.chats.application.interfaces import IMessageReader, IRoomMembership
from app.modules.chats.domain.exceptions import NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class GetMessageHistory(AuthenticatedOperation):
    room_token: str


class GetMessageHistoryUseCase:
    def __init__(self, reader: IMessageReader, membership: IRoomMembership) -> None:
        self._reader = reader
        self._membership = membership

    async def execute(self, op: GetMessageHistory) -> list[MessageDTO]:
        if not op.room_token or not await self._membership.is_member(op.room_token, op.actor.account_id):
            raise NotRoomMember()
        return await self._reader.history(op.room_token, op.actor.account_id)
