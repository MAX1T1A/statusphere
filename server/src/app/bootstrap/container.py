from dataclasses import dataclass

from asyncpg.pool import Pool

from app.modules.accounts.application.commands.issue_link_code import IssueLinkCode, IssueLinkCodeUseCase
from app.modules.accounts.application.commands.link_device import LinkDevice, LinkDeviceUseCase
from app.modules.accounts.application.commands.recover_account import RecoverAccount, RecoverAccountUseCase
from app.modules.accounts.application.commands.register_account import RegisterAccount, RegisterAccountUseCase
from app.modules.accounts.application.commands.revoke_device import RevokeDevice, RevokeDeviceUseCase
from app.modules.accounts.application.commands.set_account_name import SetAccountName, SetAccountNameUseCase
from app.modules.accounts.application.queries.list_devices import ListDevices, ListDevicesUseCase
from app.modules.accounts.infrastructure.readers import AccountReader
from app.modules.accounts.infrastructure.uow import AccountsUnitOfWork
from app.modules.chats.application.commands.send_message import SendMessage, SendMessageUseCase
from app.modules.chats.application.queries.get_history import GetMessageHistory, GetMessageHistoryUseCase
from app.modules.chats.infrastructure.readers import MessageReader
from app.modules.chats.infrastructure.uow import ChatsUnitOfWork
from app.modules.photo.application.commands.post_photo import PostPhoto, PostPhotoUseCase
from app.modules.photo.application.queries.get_photo_blob import GetPhotoBlob, GetPhotoBlobUseCase
from app.modules.photo.application.queries.list_room_photos import ListRoomPhotos, ListRoomPhotosUseCase
from app.modules.photo.infrastructure.store import PhotoStore
from app.modules.presence.application.commands.ingest_snapshot import (
    IngestPresenceSnapshot,
    IngestPresenceSnapshotUseCase,
)
from app.modules.presence.application.queries.get_hourly_activity import (
    GetHourlyActivity,
    GetHourlyActivityUseCase,
)
from app.modules.presence.application.queries.get_room_screen_time import (
    GetRoomScreenTime,
    GetRoomScreenTimeUseCase,
)
from app.modules.presence.application.queries.get_spotify_stats import GetSpotifyStats, GetSpotifyStatsUseCase
from app.modules.presence.application.queries.get_summary import GetSummary, GetSummaryUseCase
from app.modules.presence.infrastructure.readers import SnapshotReader
from app.modules.presence.infrastructure.sampler import Sampler
from app.modules.presence.infrastructure.writers import SnapshotWriter
from app.modules.realtime.infrastructure.hub import RealtimeHub
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


@dataclass
class Container:
    pool: Pool
    bus: UseCaseBus
    accounts: AccountReader
    membership: MembershipReader
    room_directory: RoomDirectory
    hub: RealtimeHub
    sampler: Sampler
    photos: PhotoStore


def build_container(pool: Pool) -> Container:
    bus = UseCaseBus()

    account_reader = AccountReader(pool)
    membership_reader = MembershipReader(pool)
    invite_codec = InviteCodec()

    hub = RealtimeHub()

    def accounts_uow() -> AccountsUnitOfWork:
        return AccountsUnitOfWork(pool)

    def rooms_uow() -> RoomsUnitOfWork:
        return RoomsUnitOfWork(pool)

    def chats_uow() -> ChatsUnitOfWork:
        return ChatsUnitOfWork(pool)

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
    bus.register(SendMessage, SendMessageUseCase(chats_uow, hub))

    interval = get_settings().sampler_interval
    snapshot_reader = SnapshotReader(pool, interval)
    bus.register(GetSummary, GetSummaryUseCase(snapshot_reader, membership_reader))
    bus.register(GetSpotifyStats, GetSpotifyStatsUseCase(snapshot_reader, membership_reader))
    bus.register(GetHourlyActivity, GetHourlyActivityUseCase(snapshot_reader, membership_reader))
    bus.register(GetRoomScreenTime, GetRoomScreenTimeUseCase(snapshot_reader, membership_reader))

    sampler = Sampler(SnapshotWriter(pool), interval)
    bus.register(IngestPresenceSnapshot, IngestPresenceSnapshotUseCase(hub, sampler))

    settings = get_settings()
    photo_store = PhotoStore(pool, settings.photos_dir, settings.photo_expiry_minutes)
    bus.register(PostPhoto, PostPhotoUseCase(photo_store, membership_reader, hub))
    bus.register(ListRoomPhotos, ListRoomPhotosUseCase(photo_store, membership_reader))
    bus.register(GetPhotoBlob, GetPhotoBlobUseCase(photo_store, membership_reader))

    return Container(
        pool=pool,
        bus=bus,
        accounts=account_reader,
        membership=membership_reader,
        room_directory=room_directory,
        hub=hub,
        sampler=sampler,
        photos=photo_store,
    )
