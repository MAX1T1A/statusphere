from typing import Callable, Optional

from app.modules.accounts.application.commands.issue_link_code import LINK_KIND
from app.modules.accounts.application.dto import AccountSessionDTO
from app.modules.accounts.application.interfaces import IAccountReader, IAccountsUnitOfWork, IRoomDirectory
from app.modules.accounts.domain.exceptions import InvalidLinkCode
from app.platform.security import generate_account_token, generate_device_id, verify_code
from app.shared_kernel.operation import Operation


class LinkDevice(Operation):
    code: str
    name: Optional[str] = None


class LinkDeviceUseCase:
    def __init__(
        self,
        reader: IAccountReader,
        uow_factory: Callable[[], IAccountsUnitOfWork],
        directory: IRoomDirectory,
    ) -> None:
        self._reader = reader
        self._uow_factory = uow_factory
        self._directory = directory

    async def execute(self, op: LinkDevice) -> AccountSessionDTO:
        subject = verify_code(LINK_KIND, op.code)
        if subject is None:
            raise InvalidLinkCode()

        parts = subject.split(":", 2)
        if len(parts) < 2:
            raise InvalidLinkCode()
        account_id, issuer_device_id = parts[0], parts[1]
        room = parts[2] if len(parts) > 2 and parts[2] else None

        if await self._reader.get_verifier(account_id) is None:
            raise InvalidLinkCode()
        if not await self._reader.is_device_active(account_id, issuer_device_id):
            raise InvalidLinkCode()

        device_id = generate_device_id()
        async with self._uow_factory() as uow:
            await uow.accounts.create_device(device_id, account_id, op.name)

        room_id = room or await self._directory.owned_room(account_id)
        return AccountSessionDTO(
            account_id=account_id,
            device_id=device_id,
            room_id=room_id or "",
            token=generate_account_token(account_id, device_id),
        )
