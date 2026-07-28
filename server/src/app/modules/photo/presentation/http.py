import base64

from fastapi import APIRouter, Depends, Query, Request, Response
from pydantic import BaseModel

from app.modules.photo.application.commands.post_photo import PostPhoto
from app.modules.photo.application.queries.get_photo_blob import GetPhotoBlob
from app.modules.photo.application.queries.list_room_photos import ListRoomPhotos
from app.modules.photo.domain.exceptions import InvalidPhoto
from app.platform.ratelimit import limit
from app.platform.web.deps import get_bus, require_actor
from app.shared_kernel.actor import Actor
from app.shared_kernel.bus import UseCaseBus

router = APIRouter(prefix="/photos", tags=["photos"])


class PostPhotoBody(BaseModel):
    room: str
    image_base64: str


@router.post("")
@limit(6)
async def post_photo(
    request: Request,
    body: PostPhotoBody,
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    try:
        image_data = base64.b64decode(body.image_base64, validate=True)
    except Exception as exc:
        raise InvalidPhoto() from exc

    photo = await bus.dispatch(PostPhoto(actor=actor, room_token=body.room, image_data=image_data))
    return photo.model_dump()


@router.get("")
@limit(30)
async def list_photos(
    request: Request,
    room: str = Query(...),
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    photos = await bus.dispatch(ListRoomPhotos(actor=actor, room_token=room))
    return {"photos": [p.model_dump() for p in photos]}


@router.get("/{account_id}/{photo_id}")
@limit(60)
async def get_photo_blob(
    request: Request,
    account_id: str,
    photo_id: str,
    room: str = Query(...),
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> Response:
    data, mime = await bus.dispatch(
        GetPhotoBlob(actor=actor, room_token=room, account_id=account_id, photo_id=photo_id)
    )
    return Response(content=data, media_type=mime, headers={"Cache-Control": "no-store"})
