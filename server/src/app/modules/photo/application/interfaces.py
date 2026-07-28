from abc import ABC, abstractmethod

from app.modules.photo.application.dto import PhotoDTO


class IPhotoStore(ABC):
    @abstractmethod
    async def save(self, room_token: str, account_id: str, image_data: bytes) -> PhotoDTO: ...

    @abstractmethod
    async def list_for_room(self, room_token: str) -> list[PhotoDTO]: ...

    @abstractmethod
    async def read_blob(self, account_id: str, photo_id: str) -> tuple[bytes, str] | None: ...


class IRoomMembership(ABC):
    @abstractmethod
    async def is_member(self, room_token: str, account_id: str) -> bool: ...


class IPhotoBroadcast(ABC):
    @abstractmethod
    async def publish_photo(
        self, room_token: str, account_id: str, photo_id: str, created_at: str, expires_at: str
    ) -> None: ...
