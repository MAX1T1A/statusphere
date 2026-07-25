def can_kick(actor_account_id: str, target_account_id: str, target_role: str | None) -> bool:
    return actor_account_id != target_account_id and target_role == "member"
