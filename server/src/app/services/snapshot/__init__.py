import logging

from app.repositories.snapshot import SnapshotRepository

from .v1.spotify_stats import spotify_stats
from .v1.summary import summary


class SnapshotService:
    def __init__(self, repository: SnapshotRepository) -> None:
        self._repository = repository

        self._logger = logging.getLogger(__name__)

    spotify_stats = spotify_stats
    summary = summary
