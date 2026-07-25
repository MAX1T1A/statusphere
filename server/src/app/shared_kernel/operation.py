from pydantic import BaseModel, ConfigDict

from app.shared_kernel.actor import Actor


class Operation(BaseModel):
    model_config = ConfigDict(frozen=True)


class AuthenticatedOperation(Operation):
    actor: Actor
