from app.platform.security import verify_account_token
from app.shared_kernel.actor import Actor
from app.shared_kernel.bus import UseCaseBus
from app.shared_kernel.exceptions import DeviceRevoked, InvalidToken
from fastapi import Header, Request


def get_bus(request: Request) -> UseCaseBus:
    return request.app.state.container.bus


async def require_actor(request: Request, x_room_token: str = Header(...)) -> Actor:
    identity = verify_account_token(x_room_token)
    if identity is None:
        raise InvalidToken()
    account_id, device_id = identity
    if not await request.app.state.container.accounts.is_device_active(account_id, device_id):
        raise DeviceRevoked()
    return Actor(account_id=account_id, device_id=device_id)
