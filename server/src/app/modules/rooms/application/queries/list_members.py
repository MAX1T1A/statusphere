from app.modules.rooms.application.dto import MemberDTO
from app.modules.rooms.application.interfaces import IMembershipReader
from app.shared_kernel.operation import AuthenticatedOperation


class ListMembers(AuthenticatedOperation):
    pass


class ListMembersUseCase:
    def __init__(self, reader: IMembershipReader) -> None:
        self._reader = reader

    async def execute(self, op: ListMembers) -> list[MemberDTO]:
        room_id = await self._reader.owned_room(op.actor.account_id)
        if room_id is None:
            return []
        rows = await self._reader.list_members(room_id)
        return [MemberDTO(**row) for row in rows]
