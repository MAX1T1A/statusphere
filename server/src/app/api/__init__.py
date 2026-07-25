from fastapi import FastAPI

from .routes.stats import router as stats_router
from .routes.ws import router as ws_router


def register_routers(app: FastAPI) -> None:
    app.include_router(stats_router)
    app.include_router(ws_router)
