from datetime import datetime

from pydantic import BaseModel


class AccountSessionDTO(BaseModel):
    account_id: str
    device_id: str
    room_id: str
    token: str


class DeviceDTO(BaseModel):
    device_id: str
    name: str | None
    revoked: bool
    linked_at: datetime
