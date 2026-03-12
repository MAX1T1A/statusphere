from app.repositories.snapshot import SnapshotRepository
from app.services.room import RoomManager
from app.services.sampler import Sampler
from app.services.snapshot import SnapshotService


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
