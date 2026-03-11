from app.detector import SystemContext

from .linux import LinuxCollector
from .linux.hyprland import HyprlandCollector
from .snapshot import Snapshot


class SystemCollector:
    def __init__(self, context: SystemContext) -> None:
        self.context = context

    def collect(self) -> Snapshot:
        snapshot = Snapshot()

        match self.context.os_family:
            case "linux":
                LinuxCollector().collect(snapshot)

                match self.context.de_wm:
                    case "hyprland":
                        HyprlandCollector().collect(snapshot)

        return snapshot
