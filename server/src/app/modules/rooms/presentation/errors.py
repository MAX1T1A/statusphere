from app.modules.rooms.domain.exceptions import InvalidOrExpiredInvite, NoRoomToInvite, NotRoomMember
from app.shared_kernel.exceptions import DomainError

ERROR_STATUS_MAP: dict[type[DomainError], int] = {
    NoRoomToInvite: 403,
    InvalidOrExpiredInvite: 400,
    NotRoomMember: 403,
}
