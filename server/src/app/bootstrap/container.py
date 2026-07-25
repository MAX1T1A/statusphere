from dataclasses import dataclass

from app.modules.chats.application.queries.get_history import GetMessageHistory, GetMessageHistoryUseCase
from app.modules.chats.infrastructure.readers import MessageReader
from app.repositories.account import AccountRepository
from app.repositories.membership import MembershipRepository
from app.shared_kernel.bus import UseCaseBus
from asyncpg.pool import Pool


@dataclass
class Container:
    pool: Pool
    bus: UseCaseBus
    accounts: AccountRepository


def build_container(pool: Pool) -> Container:
    bus = UseCaseBus()
    accounts = AccountRepository(pool)
    membership = MembershipRepository(pool)

    message_reader = MessageReader(pool)
    bus.register(GetMessageHistory, GetMessageHistoryUseCase(message_reader, membership))

    return Container(pool=pool, bus=bus, accounts=accounts)
