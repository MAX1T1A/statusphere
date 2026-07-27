import pytest

from app.modules.presence.application.queries.get_hourly_activity import (
    GetHourlyActivity,
    GetHourlyActivityUseCase,
    clamp_tz,
)
from app.modules.presence.application.queries.get_room_screen_time import (
    GetRoomScreenTime,
    GetRoomScreenTimeUseCase,
)
from app.modules.presence.application.queries.get_spotify_stats import GetSpotifyStats, GetSpotifyStatsUseCase
from app.modules.presence.application.queries.get_summary import GetSummary, GetSummaryUseCase
from app.modules.rooms.public import NotRoomMember
from app.shared_kernel.actor import Actor

ACTOR = Actor(account_id="a1", device_id="d1")


class FakeMembership:
    def __init__(self, member=True):
        self._member = member

    async def is_member(self, room, account_id):
        return self._member


class FakeReader:
    def __init__(self):
        self.summary_calls, self.spotify_calls = [], []
        self.hourly_calls, self.room_calls = [], []

    async def summary(self, room, device_id, since):
        self.summary_calls.append((room, device_id, since))
        return [{"app": "code", "seconds": 60}]

    async def spotify_aggregate(self, room, device_id, since):
        self.spotify_calls.append((room, device_id, since))
        return {"total_seconds": 120, "daily": [], "top_tracks": [], "top_artists": []}

    async def hourly_activity(self, room, device_id, tz, tz_name=""):
        self.hourly_calls.append((room, device_id, tz, tz_name))
        hours = [0] * 24
        hours[10] = 1800
        return hours

    async def room_screen_time(self, room, tz, tz_name=""):
        self.room_calls.append((room, tz, tz_name))
        return [{"account_id": "a1", "seconds": 3600}, {"account_id": "a2", "seconds": 600}]


async def test_summary_member():
    reader = FakeReader()
    uc = GetSummaryUseCase(reader, FakeMembership(member=True))
    out = await uc.execute(GetSummary(actor=ACTOR, room="R", device_id="d9", period="week"))
    assert out["device_id"] == "d9" and out["period"] == "week"
    assert out["apps"] == [{"app": "code", "seconds": 60}]
    assert reader.summary_calls and reader.summary_calls[0][0] == "R"


async def test_summary_non_member():
    uc = GetSummaryUseCase(FakeReader(), FakeMembership(member=False))
    with pytest.raises(NotRoomMember):
        await uc.execute(GetSummary(actor=ACTOR, room="R", device_id="d9"))


async def test_spotify_member():
    reader = FakeReader()
    uc = GetSpotifyStatsUseCase(reader, FakeMembership(member=True))
    out = await uc.execute(GetSpotifyStats(actor=ACTOR, room="R", device_id="d9", period="day"))
    assert out["device_id"] == "d9" and out["period"] == "day" and out["total_seconds"] == 120


async def test_spotify_non_member():
    uc = GetSpotifyStatsUseCase(FakeReader(), FakeMembership(member=False))
    with pytest.raises(NotRoomMember):
        await uc.execute(GetSpotifyStats(actor=ACTOR, room="R", device_id="d9"))


async def test_hourly_member():
    reader = FakeReader()
    uc = GetHourlyActivityUseCase(reader, FakeMembership(member=True))
    out = await uc.execute(GetHourlyActivity(actor=ACTOR, room="R", device_id="d9", tz_offset_min=300))
    assert out["device_id"] == "d9" and len(out["hours"]) == 24 and out["hours"][10] == 1800
    assert reader.hourly_calls == [("R", "d9", 300, "")]


async def test_hourly_non_member():
    uc = GetHourlyActivityUseCase(FakeReader(), FakeMembership(member=False))
    with pytest.raises(NotRoomMember):
        await uc.execute(GetHourlyActivity(actor=ACTOR, room="R", device_id="d9"))


async def test_hourly_clamps_absurd_tz():
    reader = FakeReader()
    uc = GetHourlyActivityUseCase(reader, FakeMembership(member=True))
    await uc.execute(GetHourlyActivity(actor=ACTOR, room="R", device_id="d9", tz_offset_min=100000))
    assert reader.hourly_calls[0][2] == 14 * 60


def test_clamp_tz():
    assert clamp_tz(0) == 0
    assert clamp_tz(-100000) == -14 * 60
    assert clamp_tz(330) == 330


async def test_room_screen_time_member():
    reader = FakeReader()
    uc = GetRoomScreenTimeUseCase(reader, FakeMembership(member=True))
    out = await uc.execute(GetRoomScreenTime(actor=ACTOR, room="R", tz_offset_min=-120))
    assert out["members"][0] == {"account_id": "a1", "seconds": 3600}
    assert reader.room_calls == [("R", -120, "")]


async def test_room_screen_time_non_member():
    uc = GetRoomScreenTimeUseCase(FakeReader(), FakeMembership(member=False))
    with pytest.raises(NotRoomMember):
        await uc.execute(GetRoomScreenTime(actor=ACTOR, room="R"))


async def test_hourly_passes_valid_tz_name_and_strips_garbage():
    reader = FakeReader()
    uc = GetHourlyActivityUseCase(reader, FakeMembership(member=True))
    await uc.execute(GetHourlyActivity(actor=ACTOR, room="R", device_id="d9", tz_name="Europe/Berlin"))
    await uc.execute(GetHourlyActivity(actor=ACTOR, room="R", device_id="d9", tz_name="'; DROP TABLE x;--"))
    assert reader.hourly_calls[0][3] == "Europe/Berlin"
    assert reader.hourly_calls[1][3] == ""


def test_sanitize_tz_name():
    from app.modules.presence.application.queries.get_hourly_activity import sanitize_tz_name

    assert sanitize_tz_name("Europe/Berlin") == "Europe/Berlin"
    assert sanitize_tz_name("America/Argentina/Buenos_Aires") == "America/Argentina/Buenos_Aires"
    assert sanitize_tz_name("Etc/GMT+5") == "Etc/GMT+5"
    assert sanitize_tz_name("") == ""
    assert sanitize_tz_name("bad zone; drop") == ""
    assert sanitize_tz_name("x" * 100) == ""
