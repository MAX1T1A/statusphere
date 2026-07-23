from app.core.auth.auth import verify_token
from fastapi import Header, HTTPException


async def require_auth(x_room_token: str = Header(...)) -> tuple[str, str]:
    result = verify_token(x_room_token)
    if result is None:
        raise HTTPException(401, "invalid token")
    return result
