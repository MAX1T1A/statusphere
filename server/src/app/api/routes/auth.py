from app.core.auth.auth import generate_device_id, generate_room_id, generate_token
from app.core.ratelimit.ratelimit import limit
from fastapi import APIRouter, Request
from pydantic import BaseModel

router = APIRouter(prefix="/auth", tags=["auth"])


class RegisterRequest(BaseModel):
    room_id: str | None = None


class RegisterResponse(BaseModel):
    token: str


@router.post("/register")
@limit(5)
async def register(request: Request, body: RegisterRequest) -> RegisterResponse:
    room_id = body.room_id or generate_room_id()
    device_id = generate_device_id()
    token = generate_token(room_id, device_id)
    return RegisterResponse(token=token)
