from app.modules.photo.application.interfaces import IPhotoStore, IRoomMembership
from app.modules.photo.domain.exceptions import PhotoNotFound
from app.modules.rooms.public import NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class GetPhotoBlob(AuthenticatedOperation):
    room_token: str
    account_id: str
    photo_id: str


class GetPhotoBlobUseCase:
    def __init__(self, store: IPhotoStore, membership: IRoomMembership) -> None:
        self._store = store
        self._membership = membership

    async def execute(self, op: GetPhotoBlob) -> tuple[bytes, str]:
        if not op.room_token or not await self._membership.is_member(op.room_token, op.actor.account_id):
            raise NotRoomMember()
        result = await self._store.read_blob(op.account_id, op.photo_id)
        if result is None:
            raise PhotoNotFound()
        return result
