from datetime import datetime

from pydantic import BaseModel


class MemberDTO(BaseModel):
    account_id: str
    role: str
    joined_at: datetime
