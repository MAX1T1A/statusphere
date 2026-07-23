from app.repositories.account import AccountRepository
from app.repositories.membership import MembershipRepository
from app.repositories.snapshot import SnapshotRepository
from asyncpg.pool import Pool


def provide_snapshot_repository_stub() -> SnapshotRepository:
    raise NotImplementedError


def provide_snapshot_repository(pool: Pool) -> SnapshotRepository:
    return SnapshotRepository(pool)


def provide_account_repository_stub() -> AccountRepository:
    raise NotImplementedError


def provide_account_repository(pool: Pool) -> AccountRepository:
    return AccountRepository(pool)


def provide_membership_repository_stub() -> MembershipRepository:
    raise NotImplementedError


def provide_membership_repository(pool: Pool) -> MembershipRepository:
    return MembershipRepository(pool)
