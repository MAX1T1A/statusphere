import time

from asyncpg.pool import Pool

from app.modules.healthcheck.application.interfaces import IHealthProbe

# Snapshots is a hypertable, so both queries are bounded by time on purpose:
# an unbounded max() would widen into a full scan the day timescale is absent.
_RECENT_WINDOW = "1 hour"


class HealthProbe(IHealthProbe):
    def __init__(self, pool: Pool) -> None:
        self._pool = pool

    async def database(self) -> dict:
        started = time.monotonic()
        try:
            async with self._pool.acquire() as conn:
                await conn.fetchval("SELECT 1")
        except Exception as exc:
            return {"ok": False, "error": type(exc).__name__}

        return {
            "ok": True,
            "latency_ms": round((time.monotonic() - started) * 1000, 1),
            "pool_size": self._pool.get_size(),
            "pool_idle": self._pool.get_idle_size(),
        }

    async def snapshots(self) -> dict:
        try:
            async with self._pool.acquire() as conn:
                row = await conn.fetchrow(
                    f"""
                    SELECT
                        max(created_at) AS last_at,
                        EXTRACT(EPOCH FROM now() - max(created_at)) AS age_seconds,
                        count(*) AS recent
                    FROM snapshots
                    WHERE created_at > now() - INTERVAL '{_RECENT_WINDOW}'
                    """
                )
        except Exception as exc:
            return {"ok": False, "error": type(exc).__name__}

        last_at = row["last_at"]
        age = row["age_seconds"]
        return {
            "ok": True,
            "last_at": last_at.isoformat() if last_at else None,
            # Age comes from the database clock: the server's own clock drifting
            # must not turn into a fake gap in the feed.
            "last_age_seconds": round(float(age), 1) if age is not None else None,
            "written_last_hour": row["recent"],
        }
