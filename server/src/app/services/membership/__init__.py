from app.platform.security import tokens as auth
from app.repositories.membership import MembershipRepository

INVITE_TTL = 3600


class MembershipService:
    def __init__(self, membership: MembershipRepository) -> None:
        self._membership = membership

    async def invite(self, account_id: str) -> str | None:
        room_id = await self._membership.owned_room(account_id)
        if room_id is None:
            return None
        return auth.sign_code("invite", room_id, INVITE_TTL)

    async def join(self, code: str, account_id: str) -> str | None:
        room_id = auth.verify_code("invite", code)
        if room_id is None:
            return None
        await self._membership.add_member(room_id, account_id, "member")
        return room_id

    async def members(self, account_id: str) -> list[dict]:
        room_id = await self._membership.owned_room(account_id)
        if room_id is None:
            return []
        return await self._membership.list_members(room_id)

    async def is_member(self, room_id: str, account_id: str) -> bool:
        return await self._membership.is_member(room_id, account_id)

    async def kick(self, actor_account_id: str, target_account_id: str) -> bool:
        if actor_account_id == target_account_id:
            return False
        room_id = await self._membership.owned_room(actor_account_id)
        if room_id is None:
            return False
        if await self._membership.role_of(room_id, target_account_id) != "member":
            return False
        await self._membership.remove_member(room_id, target_account_id)
        return True
