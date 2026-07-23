import app.core.ratelimit.ratelimit as rlmod
from app.core.ratelimit.ratelimit import RateLimiter


def test_allows_up_to_limit_then_blocks():
    rl = RateLimiter()
    assert rl.check("k", 2, 60) is True
    assert rl.check("k", 2, 60) is True
    assert rl.check("k", 2, 60) is False


def test_separate_keys_are_independent():
    rl = RateLimiter()
    assert rl.check("a", 1, 60) is True
    assert rl.check("b", 1, 60) is True
    assert rl.check("a", 1, 60) is False


def test_window_expiry(monkeypatch):
    now = [1000.0]
    monkeypatch.setattr(rlmod.time, "monotonic", lambda: now[0])
    rl = RateLimiter()
    assert rl.check("k", 1, 10) is True
    assert rl.check("k", 1, 10) is False
    now[0] += 11
    assert rl.check("k", 1, 10) is True
