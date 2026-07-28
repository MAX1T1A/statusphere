from app.modules.photo.application.dto import PhotoDTO
from app.modules.photo.application.interfaces import IPhotoStore, IRoomMembership
from app.modules.rooms.public import NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class ListRoomPhotos(AuthenticatedOperation):
    room_token: str


class ListRoomPhotosUseCase:
    def __init__(self, store: IPhotoStore, membership: IRoomMembership) -> None:
        self._store = store
        self._membership = membership

    async def execute(self, op: ListRoomPhotos) -> list[PhotoDTO]:
        if not op.room_token or not await self._membership.is_member(op.room_token, op.actor.account_id):
            raise NotRoomMember()
        return await self._store.list_for_room(op.room_token)
