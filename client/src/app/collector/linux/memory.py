from pathlib import Path


def _read_meminfo() -> dict[str, int]:
    result = {}
    for line in Path("/proc/meminfo").read_text().splitlines():
        key, _, value = line.partition(":")
        result[key.strip()] = int(value.strip().split()[0])
    return result


def memory_used_mb() -> float | None:
    try:
        info = _read_meminfo()
        total = info.get("MemTotal", 0)
        available = info.get("MemAvailable", 0)
        return round((total - available) / 1024, 1)
    except Exception:
        return None


def memory_total_mb() -> float | None:
    try:
        info = _read_meminfo()
        return round(info.get("MemTotal", 0) / 1024, 1)
    except Exception:
        return None
