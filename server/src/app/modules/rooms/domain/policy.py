MANAGER_ROLES = ("owner", "admin")


def can_manage(actor_role: str | None, target_role: str | None) -> bool:
    return actor_role in MANAGER_ROLES and target_role != "owner"


def can_leave(actor_role: str | None) -> bool:
    return actor_role is not None and actor_role != "owner"
