from app.shared_kernel.exceptions import DomainError


class NoRoomToInvite(DomainError):
    def __init__(self) -> None:
        super().__init__("no room to invite to")


class InvalidOrExpiredInvite(DomainError):
    def __init__(self) -> None:
        super().__init__("invalid or expired code")


class NotRoomMember(DomainError):
    def __init__(self) -> None:
        super().__init__("not a room member")
