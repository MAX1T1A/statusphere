from app.shared_kernel.exceptions import DomainError


class NotRoomMember(DomainError):
    def __init__(self) -> None:
        super().__init__("not a room member")
