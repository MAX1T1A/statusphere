from app.modules.presence.application.interfaces import ISnapshotReader
from app.modules.presence.application.queries.get_hourly_activity import clamp_tz, sanitize_tz_name
from app.modules.rooms.public import IMembershipReader, NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation


class GetRoomScreenTime(AuthenticatedOperation):
    room: str
    tz_offset_min: int = 0
    tz_name: str = ""


class GetRoomScreenTimeUseCase:
    def __init__(self, reader: ISnapshotReader, membership: IMembershipReader):
        self._reader = reader
        self._membership = membership

    async def execute(self, op: GetRoomScreenTime) -> dict:
        if not op.room or not await self._membership.is_member(op.room, op.actor.account_id):
            raise NotRoomMember()
        members = await self._reader.room_screen_time(
            op.room, clamp_tz(op.tz_offset_min), sanitize_tz_name(op.tz_name)
        )
        return {"members": members}
