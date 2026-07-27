from app.modules.rooms.application.dto import MemberDTO
from app.modules.rooms.application.interfaces import IMembershipReader
from app.modules.rooms.domain.exceptions import NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class ListMembers(AuthenticatedOperation):
    room: str


class ListMembersUseCase:
    def __init__(self, reader: IMembershipReader) -> None:
        self._reader = reader

    async def execute(self, op: ListMembers) -> list[MemberDTO]:
        if not op.room or not await self._reader.is_member(op.room, op.actor.account_id):
            raise NotRoomMember()
        rows = await self._reader.list_members(op.room)
        return [MemberDTO(**row) for row in rows]
