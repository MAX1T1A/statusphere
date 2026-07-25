from app.repositories.account import AccountRepository
from app.repositories.membership import MembershipRepository
from app.repositories.message import MessageRepository
from app.repositories.snapshot import SnapshotRepository
from app.services.account import AccountService
from app.services.membership import MembershipService
from app.services.message import MessageService
from app.services.room import RoomManager
from app.services.sampler import Sampler
from app.services.snapshot import SnapshotService


def provide_message_service(repository: MessageRepository) -> MessageService:
    return MessageService(repository)


def provide_message_service_stub() -> MessageService:
    raise NotImplementedError


def provide_room_manager() -> RoomManager:
    return RoomManager()


def provide_room_manager_stub() -> RoomManager:
    raise NotImplementedError


def provide_sampler(repository: SnapshotRepository) -> Sampler:
    return Sampler(repository)


def provide_sampler_stub() -> Sampler:
    raise NotImplementedError


def provide_snapshot_service(repository: SnapshotRepository) -> SnapshotService:
    return SnapshotService(repository)


def provide_snapshot_service_stub() -> SnapshotService:
    raise NotImplementedError


def provide_account_service(accounts: AccountRepository, membership: MembershipRepository) -> AccountService:
    return AccountService(accounts, membership)


def provide_account_service_stub() -> AccountService:
    raise NotImplementedError


def provide_membership_service(membership: MembershipRepository) -> MembershipService:
    return MembershipService(membership)


def provide_membership_service_stub() -> MembershipService:
    raise NotImplementedError
