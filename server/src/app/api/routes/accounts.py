from app.api.dependencies import require_account
from app.platform.ratelimit import limit
from app.services.account import AccountService
from app.services.providers import provide_account_service_stub
from fastapi import APIRouter, Depends, HTTPException, Request
from pydantic import BaseModel

router = APIRouter(prefix="/accounts", tags=["accounts"])


class NameRequest(BaseModel):
    name: str


class RegisterRequest(BaseModel):
    secret: str


class RecoverRequest(BaseModel):
    account_id: str
    secret: str
    name: str | None = None


class AccountResponse(BaseModel):
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
) -> AccountResponse:
    result = await service.register(body.secret)
    return AccountResponse(**result)


@router.post("/recover")
@limit(5)
async def recover(
    request: Request,
    body: RecoverRequest,
    service: AccountService = Depends(provide_account_service_stub),
) -> AccountResponse:
    result = await service.recover(body.account_id, body.secret, body.name)
    if result is None:
        raise HTTPException(401, "invalid account or secret")
    return AccountResponse(**result)


@router.post("/name")
async def set_name(
    body: NameRequest,
    auth: tuple[str, str] = Depends(require_account),
    service: AccountService = Depends(provide_account_service_stub),
) -> dict:
    account_id, _ = auth
    await service.set_name(account_id, body.name)
    return {"ok": True}
