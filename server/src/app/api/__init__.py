from fastapi import FastAPI

from .routes.ws import router as ws_router


def register_routers(app: FastAPI) -> None:
    app.include_router(ws_router)
