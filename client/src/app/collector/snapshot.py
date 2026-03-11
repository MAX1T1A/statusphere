from dataclasses import dataclass, field

from app.utils.device import get_device_id


@dataclass
class Snapshot:
    # linux-specific
    cpu_percent: float | None = None
    memory_used_mb: float | None = None
    memory_total_mb: float | None = None
    load_avg_1m: float | None = None

    # de/wm-specific
    active_workspace: str | None = None
    active_window: str | None = None
    active_app: str | None = None
