from app.core.auth.auth import generate_device_id, generate_room_id, generate_token
from fastapi import APIRouter
from pydantic import BaseModel

router = APIRouter(prefix="/auth", tags=["auth"])


class RegisterRequest(BaseModel):
    room_id: str | None = None


class RegisterResponse(BaseModel):
    token: str
    room_id: str
    device_id: str


@router.post("/register")
async def register(body: RegisterRequest) -> RegisterResponse:
    room_id = body.room_id or generate_room_id()
    device_id = generate_device_id()
    token = generate_token(room_id, device_id)
    return RegisterResponse(token=token, room_id=room_id, device_id=device_id)
