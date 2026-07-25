from typing import Callable, Optional

from app.modules.accounts.application.dto import AccountSessionDTO
from app.modules.accounts.application.interfaces import IAccountReader, IAccountsUnitOfWork, IRoomDirectory
from app.modules.accounts.domain.exceptions import InvalidCredentials
from app.platform.security import check_secret, generate_account_token, generate_device_id
from app.shared_kernel.operation import Operation


class RecoverAccount(Operation):
    account_id: str
    secret: str
    name: Optional[str] = None


class RecoverAccountUseCase:
    def __init__(
        self,
        reader: IAccountReader,
        uow_factory: Callable[[], IAccountsUnitOfWork],
        directory: IRoomDirectory,
    ) -> None:
        self._reader = reader
        self._uow_factory = uow_factory
        self._directory = directory

    async def execute(self, op: RecoverAccount) -> AccountSessionDTO:
        stored = await self._reader.get_verifier(op.account_id)
        if stored is None or not check_secret(op.secret, stored):
            raise InvalidCredentials()

        device_id = generate_device_id()
        async with self._uow_factory() as uow:
            await uow.accounts.create_device(device_id, op.account_id, op.name)

        room_id = await self._directory.owned_room(op.account_id)
        return AccountSessionDTO(
            account_id=op.account_id,
            device_id=device_id,
            room_id=room_id or "",
            token=generate_account_token(op.account_id, device_id),
        )
