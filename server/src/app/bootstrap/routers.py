from fastapi import FastAPI

from app.modules.accounts.presentation.http import accounts_router, devices_router
from app.modules.chats.presentation.http import router as chats_router
from app.modules.healthcheck.presentation.http import router as healthcheck_router
from app.modules.presence.presentation.http import router as stats_router
from app.modules.realtime.presentation.ws import router as ws_router
from app.modules.rooms.presentation.http import router as rooms_router


def register_module_routers(app: FastAPI) -> None:
    app.include_router(healthcheck_router)
    app.include_router(accounts_router)
    app.include_router(devices_router)
    app.include_router(chats_router)
    app.include_router(rooms_router)
    app.include_router(stats_router)
    app.include_router(ws_router)
