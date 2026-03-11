from app.providers import provide_room_manager_stub
from app.room import RoomManager
from fastapi import APIRouter, Depends, Header

router = APIRouter(tags=["status"])


@router.post("/status")
async def push_status(
    body: dict,
    x_room_token: str = Header(...),
    x_device_id: str = Header(...),
    room_manager: RoomManager = Depends(provide_room_manager_stub),
) -> dict:
    await room_manager.publish(x_room_token, x_device_id, body)
    return {"ok": True}
