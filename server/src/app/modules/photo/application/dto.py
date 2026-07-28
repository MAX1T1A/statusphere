from pydantic import BaseModel


class PhotoDTO(BaseModel):
    account_id: str
    photo_id: str
    created_at: str
    expires_at: str
