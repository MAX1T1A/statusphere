from app.modules.accounts.application.dto import DeviceDTO
from app.modules.accounts.application.interfaces import IAccountReader
from app.shared_kernel.operation import AuthenticatedOperation


class ListDevices(AuthenticatedOperation):
    pass


class ListDevicesUseCase:
    def __init__(self, reader: IAccountReader) -> None:
        self._reader = reader

    async def execute(self, op: ListDevices) -> list[DeviceDTO]:
        rows = await self._reader.list_devices(op.actor.account_id)
        return [DeviceDTO(**row) for row in rows]
