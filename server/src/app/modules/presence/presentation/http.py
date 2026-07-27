from fastapi import APIRouter, Depends, Query, Request

from app.modules.presence.application.queries.get_hourly_activity import GetHourlyActivity
from app.modules.presence.application.queries.get_room_screen_time import GetRoomScreenTime
from app.modules.presence.application.queries.get_spotify_stats import GetSpotifyStats
from app.modules.presence.application.queries.get_summary import GetSummary
from app.platform.ratelimit import limit
from app.platform.web.deps import get_bus, require_actor
from app.shared_kernel.actor import Actor
from app.shared_kernel.bus import UseCaseBus

router = APIRouter(prefix="/stats", tags=["stats"])


@router.get("/summary")
@limit(30)
async def summary(
    request: Request,
    room: str = Query(...),
    device_id: str = Query(...),
    period: str = Query(default="day"),
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    return await bus.dispatch(GetSummary(actor=actor, room=room, device_id=device_id, period=period))


@router.get("/spotify")
@limit(30)
async def spotify(
    request: Request,
    room: str = Query(...),
    device_id: str = Query(...),
    period: str = Query(default="week"),
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    return await bus.dispatch(GetSpotifyStats(actor=actor, room=room, device_id=device_id, period=period))


@router.get("/hourly")
@limit(30)
async def hourly(
    request: Request,
    room: str = Query(...),
    device_id: str = Query(...),
    tz_offset: int = Query(default=0),
    tz: str = Query(default=""),
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    return await bus.dispatch(
        GetHourlyActivity(actor=actor, room=room, device_id=device_id, tz_offset_min=tz_offset, tz_name=tz)
    )


@router.get("/room")
@limit(30)
async def room_screen_time(
    request: Request,
    room: str = Query(...),
    tz_offset: int = Query(default=0),
    tz: str = Query(default=""),
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    return await bus.dispatch(GetRoomScreenTime(actor=actor, room=room, tz_offset_min=tz_offset, tz_name=tz))
