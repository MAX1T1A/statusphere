from pydantic import BaseModel, ConfigDict


class Actor(BaseModel):
    model_config = ConfigDict(frozen=True)

    account_id: str
    device_id: str
