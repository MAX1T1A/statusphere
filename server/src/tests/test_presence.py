import pytest

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

    async def summary(self, room, device_id, since):
        self.summary_calls.append((room, device_id, since))
        return [{"app": "code", "seconds": 60}]

    async def spotify_aggregate(self, room, device_id, since):
        self.spotify_calls.append((room, device_id, since))
        return {"total_seconds": 120, "daily": [], "top_tracks": [], "top_artists": []}


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
