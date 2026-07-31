import time
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.bootstrap.container import build_container
from app.bootstrap.logging import configure_logging
from app.modules.realtime.presentation.ws import close_all as close_all_ws
from app.platform.db.pool import provide_pool


@asynccontextmanager
async def lifespan(app: FastAPI):
    configure_logging()
    pool = await provide_pool()
    # monotonic, so uptime survives the clock being stepped under it
    container = build_container(pool, started_at=time.monotonic(), version=app.version)
    app.state.container = container
    container.sampler.start()

    yield

    await close_all_ws()
    await container.sampler.stop()
    await pool.close()
