from datetime import datetime

import pytest

from app.modules.chats.application.commands.send_message import SendMessage, SendMessageUseCase
from app.modules.chats.application.dto import MessageDTO
from app.modules.chats.application.queries.get_history import GetMessageHistory, GetMessageHistoryUseCase
from app.modules.chats.domain.exceptions import NotRoomMember
from app.modules.chats.domain.message import MAX_TEXT, normalize_text
from app.shared_kernel.actor import Actor

ACTOR = Actor(account_id="a1", device_id="d1")


def test_normalize_text_trims_and_caps():
    assert normalize_text("  hi  ") == "hi"
    assert normalize_text("x" * 999) == "x" * MAX_TEXT
    assert normalize_text("   ") == ""


def test_message_dto_wire_shape():
    dto = MessageDTO(from_account="a", to_account="b", text="hi", at="2026-01-01T00:00:00")
    assert dto.model_dump(by_alias=True) == {"from": "a", "to": "b", "text": "hi", "at": "2026-01-01T00:00:00"}


class FakeMessages:
    def __init__(self):
        self.added = []

    async def add(self, room, frm, to, text):
        self.added.append((room, frm, to, text))
        return datetime(2026, 1, 1, 12, 0, 0)


class FakeUoW:
    def __init__(self):
        self.messages = FakeMessages()

    async def __aenter__(self):
        return self

    async def __aexit__(self, *a):
        return None


class FakeDelivery:
    def __init__(self):
        self.delivered = []

    async def deliver(self, room, frm, frm_name, to, text, at):
        self.delivered.append((room, frm, frm_name, to, text, at))


class FakeReader:
    def __init__(self, msgs):
        self._msgs = msgs

    async def history(self, room, account):
        return self._msgs


class FakeMembership:
    def __init__(self, member):
        self._member = member

    async def is_member(self, room, account):
        return self._member


async def test_send_message_persists_and_delivers():
    uow, delivery = FakeUoW(), FakeDelivery()
    uc = SendMessageUseCase(lambda: uow, delivery)
    dto = await uc.execute(SendMessage(actor=ACTOR, room_token="r", to_account="", text=" hello ", from_name="Al"))
    assert isinstance(dto, MessageDTO) and dto.text == "hello" and dto.from_account == "a1"
    assert uow.messages.added == [("r", "a1", "", "hello")]
    assert delivery.delivered and delivery.delivered[0][4] == "hello"


async def test_send_message_empty_is_noop():
    uow, delivery = FakeUoW(), FakeDelivery()
    uc = SendMessageUseCase(lambda: uow, delivery)
    result = await uc.execute(SendMessage(actor=ACTOR, room_token="r", to_account="", text="   ", from_name="Al"))
    assert result is None
    assert uow.messages.added == [] and delivery.delivered == []


async def test_get_history_requires_membership():
    uc = GetMessageHistoryUseCase(FakeReader([]), FakeMembership(False))
    with pytest.raises(NotRoomMember):
        await uc.execute(GetMessageHistory(actor=ACTOR, room_token="r"))


async def test_get_history_returns_reader_output():
    msgs = [MessageDTO(from_account="b", to_account="", text="hi", at="2026-01-01T00:00:00")]
    uc = GetMessageHistoryUseCase(FakeReader(msgs), FakeMembership(True))
    out = await uc.execute(GetMessageHistory(actor=ACTOR, room_token="r"))
    assert out == msgs
