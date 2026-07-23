from datetime import date, timedelta

from app.api.dependencies import require_auth
from app.core.ratelimit.ratelimit import limit
from app.services.providers import provide_snapshot_service_stub
from app.services.snapshot import SnapshotService
from fastapi import APIRouter, Depends, Query, Request

router = APIRouter(prefix="/stats", tags=["stats"])

PERIODS = {"day": 1, "3days": 3, "week": 7}


def since_for(period: str, default: str) -> date:
    days = PERIODS.get(period, PERIODS[default])
    return date.today() - timedelta(days=days - 1)


@router.get("/summary")
@limit(30)
async def summary(
    request: Request,
    auth: tuple[str, str] = Depends(require_auth),
    device_id: str = Query(...),
    period: str = Query(default="day"),
    service: SnapshotService = Depends(provide_snapshot_service_stub),
) -> dict:
    room_id, _ = auth
    return await service.summary(room_id, device_id, period, since_for(period, "day"))


@router.get("/spotify")
@limit(30)
async def spotify(
    request: Request,
    auth: tuple[str, str] = Depends(require_auth),
    device_id: str = Query(...),
    period: str = Query(default="week"),
    service: SnapshotService = Depends(provide_snapshot_service_stub),
) -> dict:
    room_id, _ = auth
    return await service.spotify_stats(room_id, device_id, period, since_for(period, "week"))
