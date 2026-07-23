from app.core.auth import auth
from app.repositories.account import AccountRepository
from app.repositories.membership import MembershipRepository

LINK_TTL = 300


class AccountService:
    def __init__(self, accounts: AccountRepository, membership: MembershipRepository) -> None:
        self._accounts = accounts
        self._membership = membership

    async def register(self, secret: str) -> dict:
        account_id = auth.generate_account_id()
        device_id = auth.generate_device_id()
        room_id = auth.generate_room_id()

        await self._accounts.create_account(account_id, auth.verifier(secret))
        await self._accounts.create_device(device_id, account_id, None)
        await self._membership.add_member(room_id, account_id, "owner")

        return {
            "account_id": account_id,
            "device_id": device_id,
            "room_id": room_id,
            "token": auth.generate_account_token(account_id, device_id),
        }

    async def link_code(self, account_id: str) -> str:
        return auth.sign_code("link", account_id, LINK_TTL)

    async def link_device(self, code: str, name: str | None) -> dict | None:
        account_id = auth.verify_code("link", code)
        if account_id is None:
            return None
        if await self._accounts.get_verifier(account_id) is None:
            return None

        device_id = auth.generate_device_id()
        await self._accounts.create_device(device_id, account_id, name)

        return {
            "account_id": account_id,
            "device_id": device_id,
            "room_id": await self._membership.owned_room(account_id),
            "token": auth.generate_account_token(account_id, device_id),
        }

    async def list_devices(self, account_id: str) -> list[dict]:
        return await self._accounts.list_devices(account_id)

    async def revoke_device(self, account_id: str, device_id: str) -> bool:
        return await self._accounts.revoke_device(account_id, device_id)

    async def is_device_active(self, account_id: str, device_id: str) -> bool:
        return await self._accounts.is_device_active(account_id, device_id)
