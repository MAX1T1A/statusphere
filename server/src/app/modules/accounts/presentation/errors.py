from app.modules.accounts.domain.exceptions import InvalidCredentials, InvalidLinkCode
from app.shared_kernel.exceptions import DomainError

ERROR_STATUS_MAP: dict[type[DomainError], int] = {
    InvalidCredentials: 401,
    InvalidLinkCode: 400,
}
