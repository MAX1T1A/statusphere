from pathlib import Path


def load_avg_1m() -> float | None:
    try:
        value = Path("/proc/loadavg").read_text().split()[0]
        return float(value)
    except Exception:
        return None
