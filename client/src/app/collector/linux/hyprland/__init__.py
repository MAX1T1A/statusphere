from app.collector.snapshot import Snapshot

from .window import active_app, active_window
from .workspace import active_workspace


class HyprlandCollector:
    def collect(self, snapshot: Snapshot) -> None:
        snapshot.active_window = active_window()
        snapshot.active_workspace = active_workspace()
        snapshot.active_app = active_app()
