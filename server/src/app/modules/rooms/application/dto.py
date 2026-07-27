from datetime import datetime

from pydantic import BaseModel


class MemberDTO(BaseModel):
    account_id: str
    name: str | None = None
    role: str
    joined_at: datetime
