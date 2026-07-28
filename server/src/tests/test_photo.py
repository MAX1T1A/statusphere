import pytest

from app.modules.photo.application.commands.post_photo import PostPhoto, PostPhotoUseCase
from app.modules.photo.application.dto import PhotoDTO
from app.modules.photo.application.queries.get_photo_blob import GetPhotoBlob, GetPhotoBlobUseCase
from app.modules.photo.application.queries.list_room_photos import ListRoomPhotos, ListRoomPhotosUseCase
from app.modules.photo.domain.exceptions import InvalidPhoto, PhotoNotFound
from app.modules.rooms.public import NotRoomMember
from app.shared_kernel.actor import Actor

ACTOR = Actor(account_id="a1", device_id="d1")
PHOTO = PhotoDTO(account_id="a1", photo_id="p1", created_at="2026-01-01T00:00:00+00:00", expires_at="2026-01-01T03:00:00+00:00")


class FakeStore:
    def __init__(self, saved_result=PHOTO, save_error=None, listed=None, blob=None):
        self.saved = []
        self._saved_result = saved_result
        self._save_error = save_error
        self._listed = listed if listed is not None else []
        self._blob = blob

    async def save(self, room_token, account_id, image_data):
        if self._save_error:
            raise self._save_error
        self.saved.append((room_token, account_id, image_data))
        return self._saved_result

    async def list_for_room(self, room_token):
        return self._listed

    async def read_blob(self, account_id, photo_id):
        return self._blob


class FakeMembership:
    def __init__(self, member):
        self._member = member

    async def is_member(self, room, account):
        return self._member


class FakeBroadcast:
    def __init__(self):
        self.published = []

    async def publish_photo(self, room_token, account_id, photo_id, created_at, expires_at):
        self.published.append((room_token, account_id, photo_id, created_at, expires_at))


async def test_post_photo_requires_membership():
    uc = PostPhotoUseCase(FakeStore(), FakeMembership(False), FakeBroadcast())
    with pytest.raises(NotRoomMember):
        await uc.execute(PostPhoto(actor=ACTOR, room_token="r", image_data=b"x"))


async def test_post_photo_saves_and_broadcasts():
    store, broadcast = FakeStore(), FakeBroadcast()
    uc = PostPhotoUseCase(store, FakeMembership(True), broadcast)
    dto = await uc.execute(PostPhoto(actor=ACTOR, room_token="r", image_data=b"jpeg-bytes"))
    assert dto == PHOTO
    assert store.saved == [("r", "a1", b"jpeg-bytes")]
    assert broadcast.published == [("r", "a1", "p1", PHOTO.created_at, PHOTO.expires_at)]


async def test_post_photo_invalid_image_raises_invalid_photo():
    uc = PostPhotoUseCase(FakeStore(save_error=ValueError("bad")), FakeMembership(True), FakeBroadcast())
    with pytest.raises(InvalidPhoto):
        await uc.execute(PostPhoto(actor=ACTOR, room_token="r", image_data=b"not-an-image"))


async def test_list_room_photos_requires_membership():
    uc = ListRoomPhotosUseCase(FakeStore(), FakeMembership(False))
    with pytest.raises(NotRoomMember):
        await uc.execute(ListRoomPhotos(actor=ACTOR, room_token="r"))


async def test_list_room_photos_returns_store_output():
    uc = ListRoomPhotosUseCase(FakeStore(listed=[PHOTO]), FakeMembership(True))
    out = await uc.execute(ListRoomPhotos(actor=ACTOR, room_token="r"))
    assert out == [PHOTO]


async def test_get_photo_blob_requires_membership():
    uc = GetPhotoBlobUseCase(FakeStore(), FakeMembership(False))
    with pytest.raises(NotRoomMember):
        await uc.execute(GetPhotoBlob(actor=ACTOR, room_token="r", account_id="a1", photo_id="p1"))


async def test_get_photo_blob_not_found():
    uc = GetPhotoBlobUseCase(FakeStore(blob=None), FakeMembership(True))
    with pytest.raises(PhotoNotFound):
        await uc.execute(GetPhotoBlob(actor=ACTOR, room_token="r", account_id="a1", photo_id="p1"))


async def test_get_photo_blob_returns_bytes_and_mime():
    uc = GetPhotoBlobUseCase(FakeStore(blob=(b"bytes", "image/jpeg")), FakeMembership(True))
    data, mime = await uc.execute(GetPhotoBlob(actor=ACTOR, room_token="r", account_id="a1", photo_id="p1"))
    assert data == b"bytes" and mime == "image/jpeg"
