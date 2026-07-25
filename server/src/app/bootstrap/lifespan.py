from contextlib import asynccontextmanager

from app.api.routes.ws import close_all as close_all_ws
from app.bootstrap.container import build_container
from app.bootstrap.logging import configure_logging
from app.builder import Application
from app.platform.db.pool import provide_pool
from fastapi import FastAPI


@asynccontextmanager
async def lifespan(app: FastAPI):
    configure_logging()
    pool = await provide_pool()
    app.state.container = build_container(pool)

    legacy = Application(app=app, pool=pool)
    legacy.build()
    legacy.sampler.start()
    app.state.legacy = legacy

    yield

    await close_all_ws()
    await legacy.sampler.stop()
    await pool.close()
