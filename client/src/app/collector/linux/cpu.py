import time
from pathlib import Path


def _read_cpu_stat() -> tuple[int, int]:
    line = Path("/proc/stat").read_text().splitlines()[0]
    fields = list(map(int, line.split()[1:]))
    idle = fields[3]
    total = sum(fields)
    return idle, total


def cpu_percent() -> float | None:
    try:
        idle1, total1 = _read_cpu_stat()
        time.sleep(0.1)
        idle2, total2 = _read_cpu_stat()

        total_diff = total2 - total1
        idle_diff = idle2 - idle1

        if total_diff == 0:
            return None

        return round((1 - idle_diff / total_diff) * 100, 1)
    except Exception:
        return None
