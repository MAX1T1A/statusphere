from fastapi import FastAPI

from app.bootstrap.errors import register_exception_handlers
from app.bootstrap.lifespan import lifespan
from app.bootstrap.routers import register_module_routers


def create_app() -> FastAPI:
    app = FastAPI(title="Statusphere Server", version="0.1.0", lifespan=lifespan)
    register_exception_handlers(app)
    register_module_routers(app)
    return app


app = create_app()
