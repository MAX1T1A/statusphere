from app.shared_kernel.exceptions import DomainError


class InvalidCredentials(DomainError):
    def __init__(self) -> None:
        super().__init__("invalid account or secret")


class InvalidLinkCode(DomainError):
    def __init__(self) -> None:
        super().__init__("invalid or expired code")
