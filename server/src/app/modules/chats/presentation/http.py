from fastapi import APIRouter, Depends, Query, Request

from app.modules.chats.application.queries.get_history import GetMessageHistory
from app.platform.ratelimit import limit
from app.platform.web.deps import get_bus, require_actor
from app.shared_kernel.actor import Actor
from app.shared_kernel.bus import UseCaseBus

router = APIRouter(prefix="/messages", tags=["messages"])


@router.get("")
@limit(30)
async def get_messages(
    request: Request,
    room: str = Query(...),
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    messages = await bus.dispatch(GetMessageHistory(actor=actor, room_token=room))
    return {"messages": [m.model_dump(by_alias=True) for m in messages]}
