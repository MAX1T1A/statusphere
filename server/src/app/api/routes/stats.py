from datetime import date, timedelta

from app.api.dependencies import require_account
from app.platform.ratelimit import limit
from app.services.membership import MembershipService
from app.services.providers import provide_membership_service_stub, provide_snapshot_service_stub
from app.services.snapshot import SnapshotService
from fastapi import APIRouter, Depends, HTTPException, Query, Request

router = APIRouter(prefix="/stats", tags=["stats"])

PERIODS = {"day": 1, "3days": 3, "week": 7}


def since_for(period: str, default: str) -> date:
    days = PERIODS.get(period, PERIODS[default])
    return date.today() - timedelta(days=days - 1)


async def _require_room_member(room: str, account_id: str, membership: MembershipService) -> None:
    if not room or not await membership.is_member(room, account_id):
        raise HTTPException(403, "not a room member")


@router.get("/summary")
@limit(30)
async def summary(
    request: Request,
    auth: tuple[str, str] = Depends(require_account),
    room: str = Query(...),
    device_id: str = Query(...),
    period: str = Query(default="day"),
    service: SnapshotService = Depends(provide_snapshot_service_stub),
    membership: MembershipService = Depends(provide_membership_service_stub),
) -> dict:
    account_id, _ = auth
    await _require_room_member(room, account_id, membership)
    return await service.summary(room, device_id, period, since_for(period, "day"))


@router.get("/spotify")
@limit(30)
async def spotify(
    request: Request,
    auth: tuple[str, str] = Depends(require_account),
    room: str = Query(...),
    device_id: str = Query(...),
    period: str = Query(default="week"),
    service: SnapshotService = Depends(provide_snapshot_service_stub),
    membership: MembershipService = Depends(provide_membership_service_stub),
) -> dict:
    account_id, _ = auth
    await _require_room_member(room, account_id, membership)
    return await service.spotify_stats(room, device_id, period, since_for(period, "week"))
