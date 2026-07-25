from typing import Callable

from app.modules.accounts.application.interfaces import IAccountsUnitOfWork
from app.shared_kernel.operation import AuthenticatedOperation


class RevokeDevice(AuthenticatedOperation):
    device_id: str


class RevokeDeviceUseCase:
    def __init__(self, uow_factory: Callable[[], IAccountsUnitOfWork]) -> None:
        self._uow_factory = uow_factory

    async def execute(self, op: RevokeDevice) -> bool:
        async with self._uow_factory() as uow:
            return await uow.accounts.revoke_device(op.actor.account_id, op.device_id)
