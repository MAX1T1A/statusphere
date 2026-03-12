import os

from asyncpg.pool import Pool

from .v1.save_batch import save_batch
from .v1.summary import summary


class SnapshotRepository:
    def __init__(self, pool: Pool):
        self.pool = pool
        self.sample_interval = int(os.environ.get("SAMPLER_INTERVAL", 30))

    save_batch = save_batch
    summary = summary
