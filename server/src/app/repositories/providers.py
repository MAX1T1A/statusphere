from app.repositories.snapshot import SnapshotRepository
from asyncpg.pool import Pool


def provide_snapshot_repository_stub() -> SnapshotRepository:
    raise NotImplementedError


def provide_snapshot_repository(pool: Pool) -> SnapshotRepository:
    return SnapshotRepository(pool)
