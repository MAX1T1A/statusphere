from app.collector.snapshot import Snapshot

from .cpu import cpu_percent
from .load import load_avg_1m
from .memory import memory_total_mb, memory_used_mb


class LinuxCollector:
    def collect(self, snapshot: Snapshot) -> None:
        snapshot.cpu_percent = cpu_percent()
        snapshot.memory_used_mb = memory_used_mb()
        snapshot.memory_total_mb = memory_total_mb()
        snapshot.load_avg_1m = load_avg_1m()
