import re

from app.modules.presence.application.interfaces import ISnapshotReader
from app.modules.rooms.public import IMembershipReader, NotRoomMember
from app.shared_kernel.operation import AuthenticatedOperation

MAX_TZ_OFFSET_MIN = 14 * 60

_TZ_NAME_RE = re.compile(r"^[A-Za-z][A-Za-z0-9_./+-]{0,63}$")


def clamp_tz(offset_min: int) -> int:
    return max(-MAX_TZ_OFFSET_MIN, min(MAX_TZ_OFFSET_MIN, offset_min))


def sanitize_tz_name(name: str) -> str:
    return name if _TZ_NAME_RE.match(name) else ""


class GetHourlyActivity(AuthenticatedOperation):
    room: str
    device_id: str
    tz_offset_min: int = 0
    tz_name: str = ""


class GetHourlyActivityUseCase:
    def __init__(self, reader: ISnapshotReader, membership: IMembershipReader):
        self._reader = reader
        self._membership = membership

    async def execute(self, op: GetHourlyActivity) -> dict:
        if not op.room or not await self._membership.is_member(op.room, op.actor.account_id):
            raise NotRoomMember()
        hours = await self._reader.hourly_activity(
            op.room, op.device_id, clamp_tz(op.tz_offset_min), sanitize_tz_name(op.tz_name)
        )
        return {"device_id": op.device_id, "hours": hours}
