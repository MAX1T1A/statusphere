from fastapi import FastAPI

from .routes.feed import router as feed_router
from .routes.status import router as status_router
from .routes.ws import router as ws_router


def register_routers(app: FastAPI) -> None:
    app.include_router(status_router)
    app.include_router(feed_router)
    app.include_router(ws_router)
