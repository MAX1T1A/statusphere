MAX_TEXT = 500
GROUP_LIMIT = 300
DM_LIMIT = 100


def normalize_text(raw: str) -> str:
    return raw.strip()[:MAX_TEXT]


def is_direct(to_account: str) -> bool:
    return to_account != ""
