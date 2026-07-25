from app.platform.config import get_settings
from asyncpg.pool import Pool

from .v1.save_batch import save_batch
from .v1.spotify_aggregate import spotify_aggregate
from .v1.summary import summary


class SnapshotRepository:
    def __init__(self, pool: Pool):
        self.pool = pool
        self.sample_interval = get_settings().sampler_interval

    save_batch = save_batch
    summary = summary
    spotify_aggregate = spotify_aggregate
