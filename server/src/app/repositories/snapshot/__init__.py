from asyncpg.pool import Pool

from .v1.save_batch import save_batch


class SnapshotRepository:
    def __init__(self, pool: Pool):
        self.pool = pool

    save_batch = save_batch
