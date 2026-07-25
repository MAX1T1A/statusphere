class DomainError(Exception):
    pass


class NotAuthorized(DomainError):
    pass


class InvalidToken(DomainError):
    def __init__(self) -> None:
        super().__init__("invalid token")


class DeviceRevoked(DomainError):
    def __init__(self) -> None:
        super().__init__("device revoked")
