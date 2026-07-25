from app.platform.security import verify_account_token
from app.services.account import AccountService
from app.services.providers import provide_account_service_stub
from fastapi import Depends, Header, HTTPException


async def require_account(
    x_room_token: str = Header(...),
    service: AccountService = Depends(provide_account_service_stub),
) -> tuple[str, str]:
    identity = verify_account_token(x_room_token)
    if identity is None:
        raise HTTPException(401, "invalid token")
    account_id, device_id = identity
    if not await service.is_device_active(account_id, device_id):
        raise HTTPException(401, "device revoked")
    return account_id, device_id
