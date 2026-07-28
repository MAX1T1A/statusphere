_DISPLAY_NAME = """COALESCE(NULLIF({alias}.name, ''), (
    SELECT NULLIF(d.name, '')
    FROM devices d
    WHERE d.account_id = {alias}.account_id AND NOT d.revoked AND NULLIF(d.name, '') IS NOT NULL
    ORDER BY d.linked_at
    LIMIT 1
))"""


def display_name_expr(alias: str = "a") -> str:
    """SQL for what a room sees as someone's name: the name they set, else their first device's hostname.

    Without the device fallback an account that never ran --set-name shows up as a raw id.
    """
    return _DISPLAY_NAME.format(alias=alias)
