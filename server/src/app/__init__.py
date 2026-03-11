from contextlib import asynccontextmanager

from app.builder import Application
from fastapi import FastAPI


@asynccontextmanager
async def lifespan(app: FastAPI):
    Application(app=app).build()
    yield


app = FastAPI(
    title="Statusphere Server",
    version="0.1.0",
    lifespan=lifespan,
)
