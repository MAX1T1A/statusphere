from datetime import date, timedelta

from app.services.providers import provide_snapshot_service_stub
from app.services.snapshot import SnapshotService
from fastapi import APIRouter, Depends, Query

router = APIRouter(prefix="/stats", tags=["stats"])

PERIODS = {"day": 1, "3days": 3, "week": 7}


@router.get("/summary")
async def summary(
    room_token: str = Query(...),
    device_id: str = Query(...),
    period: str = Query(default="day"),
    service: SnapshotService = Depends(provide_snapshot_service_stub),
) -> dict:
    days = PERIODS.get(period, 1)
    since = date.today() - timedelta(days=days - 1)

    return await service.summary(room_token, device_id, period, since)
