from app.api.dependencies import require_account
from app.core.ratelimit.ratelimit import limit
from app.services.account import AccountService
from app.services.providers import provide_account_service_stub
from fastapi import APIRouter, Depends, HTTPException, Request
from pydantic import BaseModel

router = APIRouter(prefix="/devices", tags=["devices"])


class LinkRequest(BaseModel):
    code: str
    name: str | None = None


class RevokeRequest(BaseModel):
    device_id: str


@router.post("/link-code")
async def link_code(
    auth: tuple[str, str] = Depends(require_account),
    service: AccountService = Depends(provide_account_service_stub),
) -> dict:
    account_id, device_id = auth
    return {"code": await service.link_code(account_id, device_id)}


@router.post("/link")
@limit(10)
async def link(
    request: Request,
    body: LinkRequest,
    service: AccountService = Depends(provide_account_service_stub),
) -> dict:
    result = await service.link_device(body.code, body.name)
    if result is None:
        raise HTTPException(400, "invalid or expired code")
    return result


@router.get("")
async def devices(
    auth: tuple[str, str] = Depends(require_account),
    service: AccountService = Depends(provide_account_service_stub),
) -> dict:
    account_id, _ = auth
    return {"devices": await service.list_devices(account_id)}


@router.post("/revoke")
async def revoke(
    body: RevokeRequest,
    auth: tuple[str, str] = Depends(require_account),
    service: AccountService = Depends(provide_account_service_stub),
) -> dict:
    account_id, _ = auth
    return {"ok": await service.revoke_device(account_id, body.device_id)}
