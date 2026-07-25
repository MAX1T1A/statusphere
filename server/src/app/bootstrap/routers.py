from app.modules.healthcheck.presentation.http import router as healthcheck_router
from fastapi import FastAPI


def register_module_routers(app: FastAPI) -> None:
    app.include_router(healthcheck_router)
