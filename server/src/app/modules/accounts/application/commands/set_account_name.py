from typing import Callable

from app.modules.accounts.application.interfaces import IAccountsUnitOfWork
from app.shared_kernel.operation import AuthenticatedOperation


class SetAccountName(AuthenticatedOperation):
    name: str


class SetAccountNameUseCase:
    def __init__(self, uow_factory: Callable[[], IAccountsUnitOfWork]) -> None:
        self._uow_factory = uow_factory

    async def execute(self, op: SetAccountName) -> None:
        async with self._uow_factory() as uow:
            await uow.accounts.set_name(op.actor.account_id, op.name)
