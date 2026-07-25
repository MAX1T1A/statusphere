from datetime import datetime

import pytest

from app.modules.accounts.application.commands.link_device import LinkDevice, LinkDeviceUseCase
from app.modules.accounts.application.commands.recover_account import RecoverAccount, RecoverAccountUseCase
from app.modules.accounts.application.commands.register_account import RegisterAccount, RegisterAccountUseCase
from app.modules.accounts.application.commands.revoke_device import RevokeDevice, RevokeDeviceUseCase
from app.modules.accounts.application.queries.list_devices import ListDevices, ListDevicesUseCase
from app.modules.accounts.domain.exceptions import InvalidCredentials, InvalidLinkCode
from app.platform.security import sign_code, verifier
from app.shared_kernel.actor import Actor

ACTOR = Actor(account_id="a1", device_id="d1")


class FakeRepo:
    def __init__(self):
        self.accounts_created, self.devices_created, self.names, self.revoked = [], [], [], []

    async def create_account(self, aid, ver):
        self.accounts_created.append((aid, ver))

    async def create_device(self, did, aid, name):
        self.devices_created.append((did, aid, name))

    async def set_name(self, aid, name):
        self.names.append((aid, name))

    async def revoke_device(self, aid, did):
        self.revoked.append((aid, did))
        return True


class FakeUoW:
    def __init__(self, repo):
        self.accounts = repo

    async def __aenter__(self):
        return self

    async def __aexit__(self, *a):
        return None


class FakeReader:
    def __init__(self, verifier_val=None, active=True, devices=None):
        self._verifier, self._active, self._devices = verifier_val, active, devices or []

    async def get_verifier(self, aid):
        return self._verifier

    async def is_device_active(self, aid, did):
        return self._active

    async def name_of(self, aid):
        return "n"

    async def list_devices(self, aid):
        return self._devices


class FakeDirectory:
    def __init__(self, room="room-x", owned="room-x"):
        self._room, self._owned = room, owned

    async def create_room_for_owner(self, aid):
        return self._room

    async def owned_room(self, aid):
        return self._owned


async def test_register_creates_account_device_room():
    repo = FakeRepo()
    uc = RegisterAccountUseCase(lambda: FakeUoW(repo), FakeDirectory(room="R"))
    dto = await uc.execute(RegisterAccount(secret="s3cret"))
    assert dto.room_id == "R" and dto.account_id and dto.device_id and dto.token
    assert len(repo.accounts_created) == 1 and len(repo.devices_created) == 1


async def test_recover_bad_secret():
    uc = RecoverAccountUseCase(FakeReader(verifier_val=None), lambda: FakeUoW(FakeRepo()), FakeDirectory())
    with pytest.raises(InvalidCredentials):
        await uc.execute(RecoverAccount(account_id="a", secret="x"))


async def test_recover_ok():
    secret = "topsecret"
    repo = FakeRepo()
    uc = RecoverAccountUseCase(FakeReader(verifier_val=verifier(secret)), lambda: FakeUoW(repo), FakeDirectory(owned="R2"))
    dto = await uc.execute(RecoverAccount(account_id="acct", secret=secret, name="laptop"))
    assert dto.account_id == "acct" and dto.room_id == "R2"
    assert repo.devices_created and repo.devices_created[0][2] == "laptop"


async def test_link_device_ok():
    code = sign_code("link", "acct1:dev1:", 300)
    repo = FakeRepo()
    uc = LinkDeviceUseCase(FakeReader(verifier_val="v", active=True), lambda: FakeUoW(repo), FakeDirectory(owned="room-y"))
    dto = await uc.execute(LinkDevice(code=code, name="phone"))
    assert dto.account_id == "acct1" and dto.room_id == "room-y"
    assert repo.devices_created and repo.devices_created[0][2] == "phone"


async def test_link_device_bad_code():
    uc = LinkDeviceUseCase(FakeReader(), lambda: FakeUoW(FakeRepo()), FakeDirectory())
    with pytest.raises(InvalidLinkCode):
        await uc.execute(LinkDevice(code="garbage"))


async def test_revoke_device():
    repo = FakeRepo()
    uc = RevokeDeviceUseCase(lambda: FakeUoW(repo))
    assert await uc.execute(RevokeDevice(actor=ACTOR, device_id="d9")) is True
    assert repo.revoked == [("a1", "d9")]


async def test_list_devices():
    reader = FakeReader(devices=[{"device_id": "d1", "name": None, "revoked": False, "linked_at": datetime(2026, 1, 1)}])
    out = await ListDevicesUseCase(reader).execute(ListDevices(actor=ACTOR))
    assert out[0].device_id == "d1"
