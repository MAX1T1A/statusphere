from datetime import date, timedelta

from app.api.routes.stats import since_for


def test_since_for_known_periods():
    assert since_for("day", "day") == date.today()
    assert since_for("3days", "day") == date.today() - timedelta(days=2)
    assert since_for("week", "day") == date.today() - timedelta(days=6)


def test_since_for_unknown_uses_default():
    assert since_for("bogus", "week") == date.today() - timedelta(days=6)
    assert since_for("bogus", "day") == date.today()
