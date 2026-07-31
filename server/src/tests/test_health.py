import time

from app.modules.healthcheck.application.queries.get_health import GetHealth, GetHealthUseCase
from app.shared_kernel.actor import Actor

ACTOR = Actor(account_id="a1", device_id="d1")


class FakeProbe:
    def __init__(self, database, snapshots):
        self._database = database
        self._snapshots = snapshots

    async def database(self):
        return self._database

    async def snapshots(self):
        return self._snapshots


def use_case(database, snapshots):
    return GetHealthUseCase(FakeProbe(database, snapshots), started_at=time.monotonic() - 60, version="1.2.3")


async def test_healthy_server_reports_ok():
    result = await use_case(
        {"ok": True, "latency_ms": 0.4},
        {"ok": True, "last_at": "2026-07-31T10:00:00+00:00", "last_age_seconds": 12.0, "written_last_hour": 240},
    ).execute(GetHealth(actor=ACTOR))

    assert result["status"] == "ok"
    assert result["version"] == "1.2.3"
    assert result["uptime_seconds"] >= 60


# The process answers, so it is not down - but a broken dependency has to be
# visible to whoever asked, not smoothed over into "ok".
async def test_broken_database_degrades_the_verdict():
    result = await use_case({"ok": False, "error": "ConnectionDoesNotExistError"}, {"ok": True}).execute(
        GetHealth(actor=ACTOR)
    )

    assert result["status"] == "degraded"
    assert result["database"]["error"] == "ConnectionDoesNotExistError"


async def test_unreadable_snapshots_degrade_the_verdict():
    result = await use_case({"ok": True}, {"ok": False, "error": "UndefinedTableError"}).execute(
        GetHealth(actor=ACTOR)
    )

    assert result["status"] == "degraded"
