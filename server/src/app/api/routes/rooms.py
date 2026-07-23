from app.api.dependencies import require_account
from app.services.membership import MembershipService
from app.services.providers import provide_membership_service_stub
from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel

router = APIRouter(prefix="/rooms", tags=["rooms"])


class JoinRequest(BaseModel):
    code: str


class KickRequest(BaseModel):
    account_id: str


@router.post("/invite")
async def invite(
    auth: tuple[str, str] = Depends(require_account),
    service: MembershipService = Depends(provide_membership_service_stub),
) -> dict:
    account_id, _ = auth
    code = await service.invite(account_id)
    if code is None:
        raise HTTPException(403, "no room to invite to")
    return {"code": code}


@router.post("/join")
async def join(
    body: JoinRequest,
    auth: tuple[str, str] = Depends(require_account),
    service: MembershipService = Depends(provide_membership_service_stub),
) -> dict:
    account_id, _ = auth
    room_id = await service.join(body.code, account_id)
    if room_id is None:
        raise HTTPException(400, "invalid or expired code")
    return {"room_id": room_id}


@router.get("/members")
async def members(
    auth: tuple[str, str] = Depends(require_account),
    service: MembershipService = Depends(provide_membership_service_stub),
) -> dict:
    account_id, _ = auth
    return {"members": await service.members(account_id)}


@router.post("/kick")
async def kick(
    body: KickRequest,
    auth: tuple[str, str] = Depends(require_account),
    service: MembershipService = Depends(provide_membership_service_stub),
) -> dict:
    account_id, _ = auth
    return {"ok": await service.kick(account_id, body.account_id)}
