from app.modules.photo.domain.exceptions import InvalidPhoto, PhotoNotFound
from app.shared_kernel.exceptions import DomainError

ERROR_STATUS_MAP: dict[type[DomainError], int] = {
    InvalidPhoto: 400,
    PhotoNotFound: 404,
}
