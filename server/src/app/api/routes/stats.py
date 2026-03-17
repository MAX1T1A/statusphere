from datetime import date, timedelta

from app.api.dependencies import require_auth
from app.core.ratelimit.ratelimit import limit
from app.services.providers import provide_snapshot_service_stub
from app.services.snapshot import SnapshotService
from fastapi import APIRouter, Depends, Query, Request

router = APIRouter(prefix="/stats", tags=["stats"])

PERIODS = {"day": 1, "3days": 3, "week": 7}


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
    days = PERIODS.get(period, 1)
    since = date.today() - timedelta(days=days - 1)

    return await service.summary(room_id, device_id, period, since)


@router.get("/spotify")
@limit(30)
async def spotify(
    auth: tuple[str, str] = Depends(require_auth),
    device_id: str = Query(...),
    period: str = Query(default="week"),
    service: SnapshotService = Depends(provide_snapshot_service_stub),
) -> dict:
    room_id, _ = auth
    days = PERIODS.get(period, 7)
    since = date.today() - timedelta(days=days - 1)

    return await service.spotify_stats(room_id, device_id, period, since)
