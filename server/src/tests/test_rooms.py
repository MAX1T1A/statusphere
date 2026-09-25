from datetime import datetime

import pytest

from app.modules.rooms.application.commands.create_invite import CreateInvite, CreateInviteUseCase
from app.modules.rooms.application.commands.join_room import JoinRoom, JoinRoomUseCase
from app.modules.rooms.application.commands.kick_member import KickMember, KickMemberUseCase
from app.modules.rooms.application.commands.leave_room import LeaveRoom, LeaveRoomUseCase
from app.modules.rooms.application.commands.set_member_role import SetMemberRole, SetMemberRoleUseCase
from app.modules.rooms.application.queries.list_members import ListMembers, ListMembersUseCase
from app.modules.rooms.domain.exceptions import InvalidOrExpiredInvite, NotRoomMember
from app.modules.rooms.domain.policy import can_leave, can_manage
from app.modules.rooms.infrastructure.invite_codec import InviteCodec
from app.shared_kernel.actor import Actor

ACTOR = Actor(account_id="owner1", device_id="d1")
ADMIN = Actor(account_id="admin1", device_id="d2")
MEMBER = Actor(account_id="member1", device_id="d3")


def test_can_manage_policy():
    assert can_manage("owner", "member") is True
    assert can_manage("owner", "admin") is True
    assert can_manage("admin", "member") is True
    assert can_manage("admin", "owner") is False
    assert can_manage("member", "member") is False
    assert can_manage(None, "member") is False


def test_can_leave_policy():
    assert can_leave("member") is True
    assert can_leave("admin") is True
    assert can_leave("owner") is False
    assert can_leave(None) is False


def test_invite_codec_round_trip():
    codec = InviteCodec()
    code = codec.sign("room-abc")
    assert codec.verify(code) == "room-abc"
    assert codec.verify("garbage") is None


class FakeReader:
    def __init__(self, managed=None, roles=None, members=None, member=True):
        self._managed, self._roles, self._members = managed, roles or {}, members or []
        self._member = member

    async def managed_room(self, account):
        return self._managed

    async def role_of(self, room, account):
        return self._roles.get(account)

    async def list_members(self, room):
        return self._members

    async def is_member(self, room, account):
        return self._member


class FakeCodec:
    def __init__(self, room=None):
        self._room = room

    def sign(self, room):
        return f"code:{room}"

    def verify(self, code):
        return self._room


class FakeMemberships:
    def __init__(self):
        self.added, self.removed, self.role_changes = [], [], []

    async def add_member(self, room, account, role="member"):
        self.added.append((room, account, role))

    async def remove_member(self, room, account):
        self.removed.append((room, account))

    async def set_role(self, room, account, role):
        self.role_changes.append((room, account, role))


class FakeUoW:
    def __init__(self):
        self.memberships = FakeMemberships()

    async def __aenter__(self):
        return self

    async def __aexit__(self, *a):
        return None


async def test_create_invite_ok():
    uc = CreateInviteUseCase(FakeReader(member=True), FakeCodec())
    assert await uc.execute(CreateInvite(actor=ACTOR, room="r1")) == "code:r1"


async def test_create_invite_not_member():
    uc = CreateInviteUseCase(FakeReader(member=False), FakeCodec())
    with pytest.raises(NotRoomMember):
        await uc.execute(CreateInvite(actor=ACTOR, room="r1"))


async def test_join_ok():
    uow = FakeUoW()
    uc = JoinRoomUseCase(lambda: uow, FakeCodec(room="r2"))
    assert await uc.execute(JoinRoom(actor=ACTOR, code="x")) == "r2"
    assert uow.memberships.added == [("r2", "owner1", "member")]


async def test_join_bad_code():
    uc = JoinRoomUseCase(lambda: FakeUoW(), FakeCodec(room=None))
    with pytest.raises(InvalidOrExpiredInvite):
        await uc.execute(JoinRoom(actor=ACTOR, code="x"))


async def test_owner_kicks_member():
    uow = FakeUoW()
    reader = FakeReader(managed="r1", roles={"owner1": "owner", "m1": "member"})
    uc = KickMemberUseCase(reader, lambda: uow)
    assert await uc.execute(KickMember(actor=ACTOR, target_account_id="m1")) is True
    assert uow.memberships.removed == [("r1", "m1")]


async def test_admin_kicks_member():
    uow = FakeUoW()
    reader = FakeReader(managed="r1", roles={"admin1": "admin", "m1": "member"})
    uc = KickMemberUseCase(reader, lambda: uow)
    assert await uc.execute(KickMember(actor=ADMIN, target_account_id="m1")) is True
    assert uow.memberships.removed == [("r1", "m1")]


async def test_admin_cannot_kick_owner():
    uow = FakeUoW()
    reader = FakeReader(managed="r1", roles={"admin1": "admin", "owner1": "owner"})
    uc = KickMemberUseCase(reader, lambda: uow)
    assert await uc.execute(KickMember(actor=ADMIN, target_account_id="owner1")) is False
    assert uow.memberships.removed == []


async def test_member_cannot_kick():
    reader = FakeReader(managed=None, roles={"member1": "member", "m1": "member"})
    uc = KickMemberUseCase(reader, lambda: FakeUoW())
    assert await uc.execute(KickMember(actor=MEMBER, target_account_id="m1")) is False


async def test_kick_self_denied():
    uc = KickMemberUseCase(FakeReader(managed="r1", roles={"owner1": "owner"}), lambda: FakeUoW())
    assert await uc.execute(KickMember(actor=ACTOR, target_account_id="owner1")) is False


async def test_leave_member_ok():
    uow = FakeUoW()
    uc = LeaveRoomUseCase(FakeReader(roles={"member1": "member"}), lambda: uow)
    assert await uc.execute(LeaveRoom(actor=MEMBER, room="r1")) is True
    assert uow.memberships.removed == [("r1", "member1")]


async def test_leave_admin_ok():
    uow = FakeUoW()
    uc = LeaveRoomUseCase(FakeReader(roles={"admin1": "admin"}), lambda: uow)
    assert await uc.execute(LeaveRoom(actor=ADMIN, room="r1")) is True
    assert uow.memberships.removed == [("r1", "admin1")]


async def test_leave_owner_denied():
    uc = LeaveRoomUseCase(FakeReader(roles={"owner1": "owner"}), lambda: FakeUoW())
    assert await uc.execute(LeaveRoom(actor=ACTOR, room="r1")) is False


async def test_leave_not_member_denied():
    uc = LeaveRoomUseCase(FakeReader(), lambda: FakeUoW())
    assert await uc.execute(LeaveRoom(actor=MEMBER, room="r1")) is False


async def test_owner_promotes_member_to_admin():
    uow = FakeUoW()
    reader = FakeReader(roles={"owner1": "owner", "m1": "member"})
    uc = SetMemberRoleUseCase(reader, lambda: uow)
    op = SetMemberRole(actor=ACTOR, room="r1", target_account_id="m1", role="admin")
    assert await uc.execute(op) is True
    assert uow.memberships.role_changes == [("r1", "m1", "admin")]


async def test_owner_demotes_admin():
    uow = FakeUoW()
    reader = FakeReader(roles={"owner1": "owner", "admin1": "admin"})
    uc = SetMemberRoleUseCase(reader, lambda: uow)
    op = SetMemberRole(actor=ACTOR, room="r1", target_account_id="admin1", role="member")
    assert await uc.execute(op) is True
    assert uow.memberships.role_changes == [("r1", "admin1", "member")]


async def test_admin_promotes_member():
    uow = FakeUoW()
    reader = FakeReader(roles={"admin1": "admin", "m1": "member"})
    uc = SetMemberRoleUseCase(reader, lambda: uow)
    op = SetMemberRole(actor=ADMIN, room="r1", target_account_id="m1", role="admin")
    assert await uc.execute(op) is True


async def test_admin_cannot_demote_owner():
    reader = FakeReader(roles={"admin1": "admin", "owner1": "owner"})
    uc = SetMemberRoleUseCase(reader, lambda: FakeUoW())
    op = SetMemberRole(actor=ADMIN, room="r1", target_account_id="owner1", role="member")
    assert await uc.execute(op) is False


async def test_member_cannot_change_roles():
    reader = FakeReader(roles={"member1": "member", "m1": "member"})
    uc = SetMemberRoleUseCase(reader, lambda: FakeUoW())
    op = SetMemberRole(actor=MEMBER, room="r1", target_account_id="m1", role="admin")
    assert await uc.execute(op) is False


async def test_demoted_admin_loses_rights():
    reader = FakeReader(managed=None, roles={"admin1": "member", "m1": "member"})
    uc = KickMemberUseCase(reader, lambda: FakeUoW())
    assert await uc.execute(KickMember(actor=ADMIN, target_account_id="m1")) is False


async def test_list_members_ok():
    members = [{"account_id": "owner1", "name": "Alice", "role": "owner", "joined_at": datetime(2026, 1, 1)}]
    uc = ListMembersUseCase(FakeReader(members=members, member=True))
    out = await uc.execute(ListMembers(actor=ACTOR, room="R"))
    assert out[0].account_id == "owner1" and out[0].name == "Alice" and out[0].role == "owner"


async def test_list_members_not_member():
    uc = ListMembersUseCase(FakeReader(member=False))
    with pytest.raises(NotRoomMember):
        await uc.execute(ListMembers(actor=ACTOR, room="R"))
