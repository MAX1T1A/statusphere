from fastapi import APIRouter, Depends, Request

from app.modules.healthcheck.application.queries.get_health import GetHealth
from app.platform.ratelimit import limit
from app.platform.web.deps import get_bus, require_actor
from app.shared_kernel.actor import Actor
from app.shared_kernel.bus import UseCaseBus

router = APIRouter(tags=["health"])


@router.get("/health")
async def health() -> dict:
    return {"status": "ok"}


# Everything past liveness is behind a device token: how many people are in the
# rooms and how the database is doing is nobody's business but the account's.
@router.get("/health/detail")
@limit(30)
async def health_detail(
    request: Request,
    actor: Actor = Depends(require_actor),
    bus: UseCaseBus = Depends(get_bus),
) -> dict:
    return await bus.dispatch(GetHealth(actor=actor))
