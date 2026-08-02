from app.modules.rooms.domain.exceptions import InvalidOrExpiredInvite, NotRoomMember
from app.shared_kernel.exceptions import DomainError

ERROR_STATUS_MAP: dict[type[DomainError], int] = {
    InvalidOrExpiredInvite: 400,
    NotRoomMember: 403,
}
