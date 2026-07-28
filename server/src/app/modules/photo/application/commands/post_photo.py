from app.modules.photo.application.dto import PhotoDTO
from app.modules.photo.application.interfaces import IPhotoBroadcast, IPhotoStore, IRoomMembership
from app.modules.photo.domain.exceptions import InvalidPhoto
from app.modules.rooms.public import NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class PostPhoto(AuthenticatedOperation):
    room_token: str
    image_data: bytes


class PostPhotoUseCase:
    def __init__(self, store: IPhotoStore, membership: IRoomMembership, broadcast: IPhotoBroadcast) -> None:
        self._store = store
        self._membership = membership
        self._broadcast = broadcast

    async def execute(self, op: PostPhoto) -> PhotoDTO:
        if not op.room_token or not await self._membership.is_member(op.room_token, op.actor.account_id):
            raise NotRoomMember()

        try:
            photo = await self._store.save(op.room_token, op.actor.account_id, op.image_data)
        except ValueError as exc:
            raise InvalidPhoto() from exc

        await self._broadcast.publish_photo(
            op.room_token, op.actor.account_id, photo.photo_id, photo.created_at, photo.expires_at
        )
        return photo
