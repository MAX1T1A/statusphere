from dataclasses import dataclass

from app.modules.chats.application.queries.get_history import GetMessageHistory, GetMessageHistoryUseCase
from app.modules.chats.infrastructure.readers import MessageReader
from app.modules.rooms.application.commands.create_invite import CreateInvite, CreateInviteUseCase
from app.modules.rooms.application.commands.join_room import JoinRoom, JoinRoomUseCase
from app.modules.rooms.application.commands.kick_member import KickMember, KickMemberUseCase
from app.modules.rooms.application.queries.list_members import ListMembers, ListMembersUseCase
from app.modules.rooms.infrastructure.invite_codec import InviteCodec
from app.modules.rooms.infrastructure.readers import MembershipReader
from app.modules.rooms.infrastructure.room_directory import RoomDirectory
from app.modules.rooms.infrastructure.uow import RoomsUnitOfWork
from app.repositories.account import AccountRepository
from app.shared_kernel.bus import UseCaseBus
from asyncpg.pool import Pool


@dataclass
class Container:
    pool: Pool
    bus: UseCaseBus
    accounts: AccountRepository
    room_directory: RoomDirectory


def build_container(pool: Pool) -> Container:
    bus = UseCaseBus()
    accounts = AccountRepository(pool)

    membership_reader = MembershipReader(pool)
    invite_codec = InviteCodec()

    def rooms_uow() -> RoomsUnitOfWork:
        return RoomsUnitOfWork(pool)

    room_directory = RoomDirectory(rooms_uow, membership_reader)

    bus.register(CreateInvite, CreateInviteUseCase(membership_reader, invite_codec))
    bus.register(JoinRoom, JoinRoomUseCase(rooms_uow, invite_codec))
    bus.register(KickMember, KickMemberUseCase(membership_reader, rooms_uow))
    bus.register(ListMembers, ListMembersUseCase(membership_reader))

    message_reader = MessageReader(pool)
    bus.register(GetMessageHistory, GetMessageHistoryUseCase(message_reader, membership_reader))

    return Container(pool=pool, bus=bus, accounts=accounts, room_directory=room_directory)
