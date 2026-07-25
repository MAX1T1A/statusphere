from dataclasses import dataclass

from app.modules.accounts.application.commands.issue_link_code import IssueLinkCode, IssueLinkCodeUseCase
from app.modules.accounts.application.commands.link_device import LinkDevice, LinkDeviceUseCase
from app.modules.accounts.application.commands.recover_account import RecoverAccount, RecoverAccountUseCase
from app.modules.accounts.application.commands.register_account import RegisterAccount, RegisterAccountUseCase
from app.modules.accounts.application.commands.revoke_device import RevokeDevice, RevokeDeviceUseCase
from app.modules.accounts.application.commands.set_account_name import SetAccountName, SetAccountNameUseCase
from app.modules.accounts.application.queries.list_devices import ListDevices, ListDevicesUseCase
from app.modules.accounts.infrastructure.readers import AccountReader
from app.modules.accounts.infrastructure.uow import AccountsUnitOfWork
from app.modules.chats.application.queries.get_history import GetMessageHistory, GetMessageHistoryUseCase
from app.modules.chats.infrastructure.readers import MessageReader
from app.modules.presence.application.queries.get_spotify_stats import GetSpotifyStats, GetSpotifyStatsUseCase
from app.modules.presence.application.queries.get_summary import GetSummary, GetSummaryUseCase
from app.modules.presence.infrastructure.readers import SnapshotReader
from app.modules.rooms.application.commands.create_invite import CreateInvite, CreateInviteUseCase
from app.modules.rooms.application.commands.join_room import JoinRoom, JoinRoomUseCase
from app.modules.rooms.application.commands.kick_member import KickMember, KickMemberUseCase
from app.modules.rooms.application.queries.list_members import ListMembers, ListMembersUseCase
from app.modules.rooms.infrastructure.invite_codec import InviteCodec
from app.modules.rooms.infrastructure.readers import MembershipReader
from app.modules.rooms.infrastructure.room_directory import RoomDirectory
from app.modules.rooms.infrastructure.uow import RoomsUnitOfWork
from app.platform.config import get_settings
from app.shared_kernel.bus import UseCaseBus
from asyncpg.pool import Pool


@dataclass
class Container:
    pool: Pool
    bus: UseCaseBus
    accounts: AccountReader
    room_directory: RoomDirectory


def build_container(pool: Pool) -> Container:
    bus = UseCaseBus()

    account_reader = AccountReader(pool)
    membership_reader = MembershipReader(pool)
    invite_codec = InviteCodec()

    def accounts_uow() -> AccountsUnitOfWork:
        return AccountsUnitOfWork(pool)

    def rooms_uow() -> RoomsUnitOfWork:
        return RoomsUnitOfWork(pool)

    room_directory = RoomDirectory(rooms_uow, membership_reader)

    bus.register(RegisterAccount, RegisterAccountUseCase(accounts_uow, room_directory))
    bus.register(RecoverAccount, RecoverAccountUseCase(account_reader, accounts_uow, room_directory))
    bus.register(SetAccountName, SetAccountNameUseCase(accounts_uow))
    bus.register(IssueLinkCode, IssueLinkCodeUseCase())
    bus.register(LinkDevice, LinkDeviceUseCase(account_reader, accounts_uow, room_directory))
    bus.register(RevokeDevice, RevokeDeviceUseCase(accounts_uow))
    bus.register(ListDevices, ListDevicesUseCase(account_reader))

    bus.register(CreateInvite, CreateInviteUseCase(membership_reader, invite_codec))
    bus.register(JoinRoom, JoinRoomUseCase(rooms_uow, invite_codec))
    bus.register(KickMember, KickMemberUseCase(membership_reader, rooms_uow))
    bus.register(ListMembers, ListMembersUseCase(membership_reader))

    message_reader = MessageReader(pool)
    bus.register(GetMessageHistory, GetMessageHistoryUseCase(message_reader, membership_reader))

    snapshot_reader = SnapshotReader(pool, get_settings().sampler_interval)
    bus.register(GetSummary, GetSummaryUseCase(snapshot_reader, membership_reader))
    bus.register(GetSpotifyStats, GetSpotifyStatsUseCase(snapshot_reader, membership_reader))

    return Container(pool=pool, bus=bus, accounts=account_reader, room_directory=room_directory)
