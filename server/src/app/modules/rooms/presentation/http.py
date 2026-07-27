from fastapi import APIRouter, Depends, Query
from pydantic import BaseModel

from app.modules.rooms.application.commands.create_invite import CreateInvite
from app.modules.rooms.application.commands.join_room import JoinRoom
from app.modules.rooms.application.commands.kick_member import KickMember
from app.modules.rooms.application.queries.list_members import ListMembers
from app.platform.web.deps import get_bus, require_actor
from app.shared_kernel.actor import Actor
from app.shared_kernel.bus import UseCaseBus

router = APIRouter(prefix="/rooms", tags=["rooms"])


class JoinRequest(BaseModel):
    code: str


class KickRequest(BaseModel):
    account_id: str


@router.post("/invite")
async def invite(actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)) -> dict:
    return {"code": await bus.dispatch(CreateInvite(actor=actor))}


@router.post("/join")
async def join(
    body: JoinRequest, actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)
) -> dict:
    return {"room_id": await bus.dispatch(JoinRoom(actor=actor, code=body.code))}


@router.get("/members")
async def members(
    room: str = Query(...), actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)
) -> dict:
    result = await bus.dispatch(ListMembers(actor=actor, room=room))
    return {"members": [m.model_dump(mode="json") for m in result]}


@router.post("/kick")
async def kick(
    body: KickRequest, actor: Actor = Depends(require_actor), bus: UseCaseBus = Depends(get_bus)
) -> dict:
    return {"ok": await bus.dispatch(KickMember(actor=actor, target_account_id=body.account_id))}
