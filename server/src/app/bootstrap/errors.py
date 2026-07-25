from app.shared_kernel.exceptions import DomainError
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

ERROR_STATUS_MAP: dict[type[DomainError], int] = {}


def register_exception_handlers(app: FastAPI) -> None:
    @app.exception_handler(DomainError)
    async def _handle_domain_error(request: Request, exc: DomainError) -> JSONResponse:
        status = ERROR_STATUS_MAP.get(type(exc), 400)
        return JSONResponse(status_code=status, content={"detail": str(exc)})
