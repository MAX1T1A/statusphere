from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

from app.modules.accounts.presentation.errors import ERROR_STATUS_MAP as ACCOUNTS_ERRORS
from app.modules.chats.presentation.errors import ERROR_STATUS_MAP as CHATS_ERRORS
from app.modules.photo.presentation.errors import ERROR_STATUS_MAP as PHOTO_ERRORS
from app.modules.rooms.presentation.errors import ERROR_STATUS_MAP as ROOMS_ERRORS
from app.shared_kernel.exceptions import DeviceRevoked, DomainError, InvalidToken

ERROR_STATUS_MAP: dict[type[DomainError], int] = {
    InvalidToken: 401,
    DeviceRevoked: 401,
    **ACCOUNTS_ERRORS,
    **CHATS_ERRORS,
    **ROOMS_ERRORS,
    **PHOTO_ERRORS,
}


def register_exception_handlers(app: FastAPI) -> None:
    @app.exception_handler(DomainError)
    async def _handle_domain_error(request: Request, exc: DomainError) -> JSONResponse:
        status = ERROR_STATUS_MAP.get(type(exc), 400)
        return JSONResponse(status_code=status, content={"detail": str(exc)})
