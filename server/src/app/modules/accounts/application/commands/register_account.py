from typing import Callable

from app.modules.accounts.application.dto import AccountSessionDTO
from app.modules.accounts.application.interfaces import IAccountsUnitOfWork, IRoomDirectory
from app.platform.security import generate_account_id, generate_account_token, generate_device_id, verifier
from app.shared_kernel.operation import Operation


class RegisterAccount(Operation):
    secret: str
    name: str | None = None


class RegisterAccountUseCase:
    def __init__(self, uow_factory: Callable[[], IAccountsUnitOfWork], directory: IRoomDirectory) -> None:
        self._uow_factory = uow_factory
        self._directory = directory

    async def execute(self, op: RegisterAccount) -> AccountSessionDTO:
        account_id = generate_account_id()
        device_id = generate_device_id()

        async with self._uow_factory() as uow:
            await uow.accounts.create_account(account_id, verifier(op.secret))
            await uow.accounts.create_device(device_id, account_id, op.name)

        room_id = await self._directory.create_room_for_owner(account_id)
        return AccountSessionDTO(
            account_id=account_id,
            device_id=device_id,
            room_id=room_id,
            token=generate_account_token(account_id, device_id),
        )
