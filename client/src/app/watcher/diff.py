from app.collector.snapshot import Snapshot


def has_diff(a: Snapshot, b: Snapshot) -> bool:
    return a != b
