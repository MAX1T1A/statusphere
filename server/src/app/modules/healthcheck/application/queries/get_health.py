import time

from app.modules.healthcheck.application.interfaces import IHealthProbe
from app.shared_kernel.operation import AuthenticatedOperation


class GetHealth(AuthenticatedOperation):
    pass


class GetHealthUseCase:
    def __init__(self, probe: IHealthProbe, started_at: float, version: str):
        self._probe = probe
        self._started_at = started_at
        self._version = version

    async def execute(self, op: GetHealth) -> dict:
        database = await self._probe.database()
        snapshots = await self._probe.snapshots()

        return {
            # Degraded, not down: the process answering this is by definition up,
            # and what is broken is whatever it depends on.
            "status": "ok" if database.get("ok") and snapshots.get("ok") else "degraded",
            "version": self._version,
            "uptime_seconds": round(time.monotonic() - self._started_at, 1),
            "database": database,
            "snapshots": snapshots,
        }
