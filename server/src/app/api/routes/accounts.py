from app.core.ratelimit.ratelimit import limit
from app.services.account import AccountService
from app.services.providers import provide_account_service_stub
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel

router = APIRouter(prefix="/accounts", tags=["accounts"])


class RegisterRequest(BaseModel):
    secret: str


class RegisterResponse(BaseModel):
    account_id: str
    device_id: str
    room_id: str
    token: str


@router.post("/register")
@limit(5)
async def register(
    request: Request,
    body: RegisterRequest,
    service: AccountService = Depends(provide_account_service_stub),
) -> RegisterResponse:
    result = await service.register(body.secret)
    return RegisterResponse(**result)
