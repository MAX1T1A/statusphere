from app.api.dependencies import require_account
from app.platform.ratelimit import limit
from app.services.membership import MembershipService
from app.services.message import MessageService
from app.services.providers import provide_membership_service_stub, provide_message_service_stub
from fastapi import APIRouter, Depends, HTTPException, Query, Request

router = APIRouter(prefix="/messages", tags=["messages"])


@router.get("")
@limit(30)
async def history(
    request: Request,
    auth: tuple[str, str] = Depends(require_account),
    room: str = Query(...),
    service: MessageService = Depends(provide_message_service_stub),
    membership: MembershipService = Depends(provide_membership_service_stub),
) -> dict:
    account_id, _ = auth
    if not room or not await membership.is_member(room, account_id):
        raise HTTPException(403, "not a room member")
    return {"messages": await service.history(room, account_id)}
