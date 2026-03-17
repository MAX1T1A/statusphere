from app.core.auth.auth import verify_token
from fastapi import Header, HTTPException, WebSocket


async def require_auth(x_room_token: str = Header(...)) -> tuple[str, str]:
    result = verify_token(x_room_token)
    if result is None:
        raise HTTPException(403, "invalid token")
    return result


async def ws_auth(websocket: WebSocket) -> tuple[str, str] | None:
    token = websocket.headers.get("x-room-token")
    if not token:
        return None
    return verify_token(token)
