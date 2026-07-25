from app.modules.chats.domain.exceptions import NotRoomMember
from app.shared_kernel.exceptions import DomainError

ERROR_STATUS_MAP: dict[type[DomainError], int] = {
    NotRoomMember: 403,
}
