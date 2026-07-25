from app.modules.accounts.application.commands.issue_link_code import IssueLinkCode
from app.modules.accounts.application.commands.link_device import LinkDevice
from app.modules.accounts.application.commands.recover_account import RecoverAccount
from app.modules.accounts.application.commands.register_account import RegisterAccount
from app.modules.accounts.application.commands.revoke_device import RevokeDevice
from app.modules.accounts.application.commands.set_account_name import SetAccountName
from app.modules.accounts.application.dto import AccountSessionDTO
from app.modules.accounts.application.queries.list_devices import ListDevices
from app.platform.ratelimit import limit
from app.platform.web.deps import get_bus, require_actor
from app.shared_kernel.actor import Actor
from app.shared_kernel.bus import UseCaseBus
from fastapi import APIRouter, Depends, Request
from pydantic import BaseModel

accounts_router = APIRouter(prefix="/accounts", tags=["accounts"])
devices_router = APIRouter(prefix="/devices", tags=["devices"])


class RegisterRequest(BaseModel):
    secret: str


class RecoverRequest(BaseModel):
    account_id: str
    secret: str
    name: str | None = None


class NameRequest(BaseModel):
    name: str


class LinkCodeRequest(BaseModel):
    room: str = ""


class LinkRequest(BaseModel):
    code: str
    name: str | None = None


class RevokeRequest(BaseModel):
    device_id: str


@accounts_router.post("/register")
@limit(5)
async def register(request: Request, body: RegisterRequest, bus: UseCaseBus = Depends(get_bus)) -> AccountSessionDTO:
    return await bus.dispatch(RegisterAccount(secret=body.secret))


@accounts_router.post("/recover")
@limit(5)
async def recover(request: Request, body: RecoverRequest, bus: UseCaseBus = Depends(get_bus)) -> AccountSessionDTO:
    return await bus.dispatch(RecoverAccount(account_id=body.account_id, secret=body.secret, name=body.name))


@accounts_router.post("/name")
async def set_name(
    body: NameRequest, actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)
) -> dict:
    await bus.dispatch(SetAccountName(actor=actor, name=body.name))
    return {"ok": True}


@devices_router.post("/link-code")
async def link_code(
    body: LinkCodeRequest, actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)
) -> dict:
    return {"code": await bus.dispatch(IssueLinkCode(actor=actor, room=body.room))}


@devices_router.post("/link")
@limit(10)
async def link(request: Request, body: LinkRequest, bus: UseCaseBus = Depends(get_bus)) -> AccountSessionDTO:
    return await bus.dispatch(LinkDevice(code=body.code, name=body.name))


@devices_router.get("")
async def devices(actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)) -> dict:
    result = await bus.dispatch(ListDevices(actor=actor))
    return {"devices": [d.model_dump(mode="json") for d in result]}


@devices_router.post("/revoke")
async def revoke(
    body: RevokeRequest, actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)
) -> dict:
    return {"ok": await bus.dispatch(RevokeDevice(actor=actor, device_id=body.device_id))}
