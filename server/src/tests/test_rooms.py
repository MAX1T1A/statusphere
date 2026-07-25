import pytest

from app.modules.rooms.application.commands.create_invite import CreateInvite, CreateInviteUseCase
from app.modules.rooms.application.commands.join_room import JoinRoom, JoinRoomUseCase
from app.modules.rooms.application.commands.kick_member import KickMember, KickMemberUseCase
from app.modules.rooms.application.queries.list_members import ListMembers, ListMembersUseCase
from app.modules.rooms.domain.exceptions import InvalidOrExpiredInvite, NoRoomToInvite
from app.modules.rooms.domain.policy import can_kick
from app.modules.rooms.infrastructure.invite_codec import InviteCodec
from app.shared_kernel.actor import Actor

ACTOR = Actor(account_id="owner1", device_id="d1")


def test_can_kick_policy():
    assert can_kick("o", "m", "member") is True
    assert can_kick("o", "o", "member") is False
    assert can_kick("o", "m", "owner") is False
    assert can_kick("o", "m", None) is False


def test_invite_codec_round_trip():
    codec = InviteCodec()
    code = codec.sign("room-abc")
    assert codec.verify(code) == "room-abc"
    assert codec.verify("garbage") is None


class FakeReader:
    def __init__(self, owned=None, role=None, members=None):
        self._owned, self._role, self._members = owned, role, members or []

    async def owned_room(self, account):
        return self._owned

    async def role_of(self, room, account):
        return self._role

    async def list_members(self, room):
        return self._members

    async def is_member(self, room, account):
        return True


class FakeCodec:
    def __init__(self, room=None):
        self._room = room

    def sign(self, room):
        return f"code:{room}"

    def verify(self, code):
        return self._room


class FakeMemberships:
    def __init__(self):
        self.added, self.removed = [], []

    async def add_member(self, room, account, role="member"):
        self.added.append((room, account, role))

    async def remove_member(self, room, account):
        self.removed.append((room, account))


class FakeUoW:
    def __init__(self):
        self.memberships = FakeMemberships()

    async def __aenter__(self):
        return self

    async def __aexit__(self, *a):
        return None


async def test_create_invite_ok():
    uc = CreateInviteUseCase(FakeReader(owned="r1"), FakeCodec())
    assert await uc.execute(CreateInvite(actor=ACTOR)) == "code:r1"


async def test_create_invite_no_room():
    uc = CreateInviteUseCase(FakeReader(owned=None), FakeCodec())
    with pytest.raises(NoRoomToInvite):
        await uc.execute(CreateInvite(actor=ACTOR))


async def test_join_ok():
    uow = FakeUoW()
    uc = JoinRoomUseCase(lambda: uow, FakeCodec(room="r2"))
    assert await uc.execute(JoinRoom(actor=ACTOR, code="x")) == "r2"
    assert uow.memberships.added == [("r2", "owner1", "member")]


async def test_join_bad_code():
    uc = JoinRoomUseCase(lambda: FakeUoW(), FakeCodec(room=None))
    with pytest.raises(InvalidOrExpiredInvite):
        await uc.execute(JoinRoom(actor=ACTOR, code="x"))


async def test_kick_member_ok():
    uow = FakeUoW()
    uc = KickMemberUseCase(FakeReader(owned="r1", role="member"), lambda: uow)
    assert await uc.execute(KickMember(actor=ACTOR, target_account_id="m1")) is True
    assert uow.memberships.removed == [("r1", "m1")]


async def test_kick_self_denied():
    uc = KickMemberUseCase(FakeReader(owned="r1", role="member"), lambda: FakeUoW())
    assert await uc.execute(KickMember(actor=ACTOR, target_account_id="owner1")) is False


async def test_list_members_no_room():
    uc = ListMembersUseCase(FakeReader(owned=None))
    assert await uc.execute(ListMembers(actor=ACTOR)) == []
