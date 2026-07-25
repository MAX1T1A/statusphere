from app.modules.presence.application.interfaces import ISnapshotReader
from app.modules.presence.application.period import since_for
from app.modules.rooms.public import IMembershipReader, NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class GetSpotifyStats(AuthenticatedOperation):
    room: str
    device_id: str
    period: str = "week"


class GetSpotifyStatsUseCase:
    def __init__(self, reader: ISnapshotReader, membership: IMembershipReader):
        self._reader = reader
        self._membership = membership

    async def execute(self, op: GetSpotifyStats) -> dict:
        if not op.room or not await self._membership.is_member(op.room, op.actor.account_id):
            raise NotRoomMember()
        since = since_for(op.period, "week")
        aggregate = await self._reader.spotify_aggregate(op.room, op.device_id, since)
        return {"device_id": op.device_id, "period": op.period, "since": str(since), **aggregate}
