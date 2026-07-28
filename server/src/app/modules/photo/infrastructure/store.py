import io
import os
import uuid
from datetime import datetime, timedelta, timezone

from asyncpg.pool import Pool
from PIL import Image

from app.modules.photo.application.dto import PhotoDTO
from app.modules.photo.application.interfaces import IPhotoStore
from app.modules.photo.domain.photo import ALLOWED_INPUT_FORMATS, MAX_DIMENSION, MAX_UPLOAD_BYTES, OUTPUT_QUALITY
from app.platform.crypto import decrypt, encrypt

_UPSERT = """
    INSERT INTO photo_shares (account_id, room_token, photo_id, mime, width, height, byte_size,
        created_at, expires_at)
    VALUES ($1, $2, $3, 'image/jpeg', $4, $5, $6, $7, $8)
    ON CONFLICT (account_id) DO UPDATE SET
        room_token = $2, photo_id = $3, mime = 'image/jpeg', width = $4, height = $5, byte_size = $6,
        created_at = $7, expires_at = $8
"""


class PhotoStore(IPhotoStore):
    def __init__(self, pool: Pool, photos_dir: str, expiry_minutes: int) -> None:
        self._pool = pool
        self._dir = photos_dir
        self._expiry_minutes = expiry_minutes
        os.makedirs(self._dir, exist_ok=True)

    def _path(self, account_id: str) -> str:
        return os.path.join(self._dir, f"{account_id}.enc")

    async def save(self, room_token: str, account_id: str, image_data: bytes) -> PhotoDTO:
        if not image_data or len(image_data) > MAX_UPLOAD_BYTES:
            raise ValueError("photo too large")

        try:
            probe = Image.open(io.BytesIO(image_data))
            probe.verify()
            image = Image.open(io.BytesIO(image_data))  # verify() consumes the handle, reopen
        except Exception as exc:
            raise ValueError("not a valid image") from exc

        if image.format not in ALLOWED_INPUT_FORMATS:
            raise ValueError("unsupported image format")

        # Re-encoding through a fresh JPEG drops EXIF (including GPS) along the way.
        image = image.convert("RGB")
        image.thumbnail((MAX_DIMENSION, MAX_DIMENSION))

        buf = io.BytesIO()
        image.save(buf, format="JPEG", quality=OUTPUT_QUALITY)
        encoded = buf.getvalue()

        photo_id = uuid.uuid4().hex
        created_at = datetime.now(timezone.utc)
        expires_at = created_at + timedelta(minutes=self._expiry_minutes)

        tmp_path = self._path(account_id) + ".tmp"
        with open(tmp_path, "wb") as f:
            f.write(encrypt(encoded))
        os.replace(tmp_path, self._path(account_id))

        async with self._pool.acquire() as conn:
            await conn.execute(
                _UPSERT,
                account_id,
                room_token,
                photo_id,
                image.width,
                image.height,
                len(encoded),
                created_at,
                expires_at,
            )

        return PhotoDTO(
            account_id=account_id,
            photo_id=photo_id,
            created_at=created_at.isoformat(),
            expires_at=expires_at.isoformat(),
        )

    async def list_for_room(self, room_token: str) -> list[PhotoDTO]:
        async with self._pool.acquire() as conn:
            rows = await conn.fetch(
                "SELECT account_id, photo_id, created_at, expires_at FROM photo_shares "
                "WHERE room_token = $1 AND expires_at > now()",
                room_token,
            )
        return [
            PhotoDTO(
                account_id=r["account_id"],
                photo_id=r["photo_id"],
                created_at=r["created_at"].isoformat(),
                expires_at=r["expires_at"].isoformat(),
            )
            for r in rows
        ]

    async def read_blob(self, account_id: str, photo_id: str) -> tuple[bytes, str] | None:
        async with self._pool.acquire() as conn:
            row = await conn.fetchrow(
                "SELECT photo_id, mime FROM photo_shares WHERE account_id = $1 AND expires_at > now()",
                account_id,
            )
        if row is None or row["photo_id"] != photo_id:
            return None
        try:
            with open(self._path(account_id), "rb") as f:
                blob = f.read()
        except FileNotFoundError:
            return None
        return decrypt(blob), row["mime"]
