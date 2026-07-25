from fastapi import FastAPI

from .routes.accounts import router as accounts_router
from .routes.devices import router as devices_router
from .routes.stats import router as stats_router
from .routes.ws import router as ws_router


def register_routers(app: FastAPI) -> None:
    app.include_router(accounts_router)
    app.include_router(devices_router)
    app.include_router(stats_router)
    app.include_router(ws_router)
