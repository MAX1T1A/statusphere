from datetime import date, timedelta

PERIODS = {"day": 1, "3days": 3, "week": 7}


def since_for(period: str, default: str) -> date:
    days = PERIODS.get(period, PERIODS[default])
    return date.today() - timedelta(days=days - 1)
