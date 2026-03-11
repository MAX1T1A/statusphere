import json

from app.providers import provide_room_manager_stub
from app.room import RoomManager
from fastapi import APIRouter, Depends, Header
from fastapi.responses import StreamingResponse

router = APIRouter(tags=["feed"])


@router.get("/feed")
async def get_feed(
    x_room_token: str = Header(...),
    x_device_id: str = Header(...),
    room_manager: RoomManager = Depends(provide_room_manager_stub),
) -> StreamingResponse:
    async def event_stream():
        async for data in room_manager.subscribe(x_room_token, x_device_id):
            yield f"data: {json.dumps(data)}\n\n"

    return StreamingResponse(event_stream(), media_type="text/event-stream")
