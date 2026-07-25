from app.modules.presence.application.interfaces import ISnapshotReader
from app.modules.presence.application.period import since_for
from app.modules.rooms.public import IMembershipReader, NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class GetSummary(AuthenticatedOperation):
    room: str
    device_id: str
    period: str = "day"


class GetSummaryUseCase:
    def __init__(self, reader: ISnapshotReader, membership: IMembershipReader):
        self._reader = reader
        self._membership = membership

    async def execute(self, op: GetSummary) -> dict:
        if not op.room or not await self._membership.is_member(op.room, op.actor.account_id):
            raise NotRoomMember()
        since = since_for(op.period, "day")
        apps = await self._reader.summary(op.room, op.device_id, since)
        return {"device_id": op.device_id, "period": op.period, "since": str(since), "apps": apps}
