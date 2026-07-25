from contextlib import asynccontextmanager

from app.bootstrap.container import build_container
from app.bootstrap.logging import configure_logging
from app.modules.realtime.presentation.ws import close_all as close_all_ws
from app.platform.db.pool import provide_pool
from fastapi import FastAPI


@asynccontextmanager
async def lifespan(app: FastAPI):
    configure_logging()
    pool = await provide_pool()
    container = build_container(pool)
    app.state.container = container
    container.sampler.start()

    yield

    await close_all_ws()
    await container.sampler.stop()
    await pool.close()
