from app.core.auth import auth
from app.services.account import AccountService
from app.services.membership import MembershipService


class FakeAccountRepo:
    def __init__(self):
        self.accounts = {}
        self.devices = {}

    async def create_account(self, account_id, secret_verifier):
        self.accounts[account_id] = secret_verifier

    async def get_verifier(self, account_id):
        return self.accounts.get(account_id)

    async def create_device(self, device_id, account_id, name):
        self.devices[device_id] = {"account_id": account_id, "name": name, "revoked": False}

    async def list_devices(self, account_id):
        return [{"device_id": d, **v} for d, v in self.devices.items() if v["account_id"] == account_id]

    async def revoke_device(self, account_id, device_id):
        dev = self.devices.get(device_id)
        if dev and dev["account_id"] == account_id:
            dev["revoked"] = True
            return True
        return False

    async def is_device_active(self, account_id, device_id):
        dev = self.devices.get(device_id)
        return dev is not None and dev["account_id"] == account_id and not dev["revoked"]


class FakeMembershipRepo:
    def __init__(self):
        self.members = {}

    async def add_member(self, room_id, account_id, role="member"):
        self.members.setdefault(room_id, {}).setdefault(account_id, role)

    async def remove_member(self, room_id, account_id):
        self.members.get(room_id, {}).pop(account_id, None)

    async def is_member(self, room_id, account_id):
        return account_id in self.members.get(room_id, {})

    async def list_members(self, room_id):
        return [{"account_id": a, "role": r} for a, r in self.members.get(room_id, {}).items()]

    async def role_of(self, room_id, account_id):
        return self.members.get(room_id, {}).get(account_id)

    async def owned_room(self, account_id):
        for room_id, members in self.members.items():
            if members.get(account_id) == "owner":
                return room_id
        return None


def _service():
    accounts, membership = FakeAccountRepo(), FakeMembershipRepo()
    return AccountService(accounts, membership), MembershipService(membership), accounts, membership


async def test_register_creates_account_device_and_owner_room():
    acc, _, accounts, membership = _service()
    result = await acc.register("my-secret")

    assert auth.verify_account_token(result["token"]) == (result["account_id"], result["device_id"])
    assert auth.check_secret("my-secret", accounts.accounts[result["account_id"]])
    assert await membership.owned_room(result["account_id"]) == result["room_id"]


async def test_link_device_adds_device_under_same_account():
    acc, _, _, _ = _service()
    reg = await acc.register("s")
    code = await acc.link_code(reg["account_id"], reg["device_id"])

    linked = await acc.link_device(code, "phone")
    assert linked is not None
    assert linked["account_id"] == reg["account_id"]
    assert linked["device_id"] != reg["device_id"]
    assert linked["room_id"] == reg["room_id"]
    assert auth.verify_account_token(linked["token"]) == (reg["account_id"], linked["device_id"])


async def test_link_rejects_expired_and_unknown():
    acc, _, _, _ = _service()
    assert await acc.link_device(auth.sign_code("link", "acc:dev", -1), None) is None
    assert await acc.link_device(auth.sign_code("link", "ghost:dev", 300), None) is None


async def test_link_rejects_code_from_revoked_issuer():
    acc, _, _, _ = _service()
    reg = await acc.register("s")
    code = await acc.link_code(reg["account_id"], reg["device_id"])
    await acc.revoke_device(reg["account_id"], reg["device_id"])
    assert await acc.link_device(code, "phone") is None


async def test_revoke_makes_device_inactive():
    acc, _, _, _ = _service()
    reg = await acc.register("s")
    assert await acc.is_device_active(reg["account_id"], reg["device_id"]) is True
    assert await acc.revoke_device(reg["account_id"], reg["device_id"]) is True
    assert await acc.is_device_active(reg["account_id"], reg["device_id"]) is False


async def test_invite_join_members_and_kick():
    acc, mem, _, _ = _service()
    owner = await acc.register("owner-secret")
    guest = await acc.register("guest-secret")

    code = await mem.invite(owner["account_id"])
    assert code is not None
    assert await mem.join(code, guest["account_id"]) == owner["room_id"]

    members = await mem.members(owner["account_id"])
    ids = {m["account_id"] for m in members}
    assert owner["account_id"] in ids and guest["account_id"] in ids

    assert await mem.kick(guest["account_id"], owner["account_id"]) is False
    assert await mem.kick(owner["account_id"], guest["account_id"]) is True
    remaining = {m["account_id"] for m in await mem.members(owner["account_id"])}
    assert guest["account_id"] not in remaining
