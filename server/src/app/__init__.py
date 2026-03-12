from contextlib import asynccontextmanager

from app.builder import Application
from app.db.connection import provide_pool
from fastapi import FastAPI


@asynccontextmanager
async def lifespan(app: FastAPI):
    pool = await provide_pool()
    application = Application(app=app, pool=pool)
    application.build()
    application.sampler.start()

    yield

    await application.sampler.stop()
    await pool.close()


app = FastAPI(
    title="Statusphere Server",
    version="0.1.0",
    lifespan=lifespan,
)
