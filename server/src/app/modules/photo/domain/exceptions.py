from app.shared_kernel.exceptions import DomainError


class InvalidPhoto(DomainError):
    def __init__(self) -> None:
        super().__init__("invalid photo")


class PhotoNotFound(DomainError):
    def __init__(self) -> None:
        super().__init__("photo not found")
