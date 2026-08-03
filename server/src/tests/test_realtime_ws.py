from dataclasses import dataclass, field

import pytest
from fastapi import FastAPI
from starlette.testclient import TestClient
from starlette.websockets import WebSocketDisconnect

from app.modules.realtime.presentation.ws import router as ws_router
from app.platform.security.tokens import generate_account_token

ACCOUNT_ID = "acct-1"
DEVICE_ID = "device-1"
ROOM_ID = "room-1"


class FakeAccounts:
    def __init__(self, active=True):
        self._active = active

    async def is_device_active(self, account_id, device_id):
        return self._active

    async def name_of(self, account_id):
        return "Someone"


class FakeMembership:
    def __init__(self, member=True):
        self._member = member

    async def is_member(self, room_id, account_id):
        return self._member


class FakeBus:
    def __init__(self):
        self.dispatched = []

    async def dispatch(self, command):
        self.dispatched.append(command)


class FakeHub:
    async def subscribe(self, room, device_id, account_id):
        return
        yield  # pragma: no cover - makes this an async generator that never yields


@dataclass
class FakeContainer:
    accounts: FakeAccounts
    membership: FakeMembership
    bus: FakeBus = field(default_factory=FakeBus)
    hub: FakeHub = field(default_factory=FakeHub)


def make_client(accounts=None, membership=None):
    app = FastAPI()
    app.include_router(ws_router)
    app.state.container = FakeContainer(
        accounts=accounts or FakeAccounts(),
        membership=membership or FakeMembership(),
    )
    return TestClient(app)


def token():
    return generate_account_token(ACCOUNT_ID, DEVICE_ID)


def test_rejects_invalid_token():
    client = make_client()
    with pytest.raises(WebSocketDisconnect):
        with client.websocket_connect(f"/ws?room={ROOM_ID}", headers={"x-room-token": "garbage"}) as ws:
            ws.receive_text()


def test_rejects_revoked_device():
    client = make_client(accounts=FakeAccounts(active=False))
    with pytest.raises(WebSocketDisconnect):
        with client.websocket_connect(f"/ws?room={ROOM_ID}", headers={"x-room-token": token()}) as ws:
            ws.receive_text()


def test_rejects_when_room_id_does_not_match_membership():
    # This is the exact production incident: the client had a valid, non-revoked
    # token but pointed at a room the account isn't a member of. ws.py closes
    # before accept() in this case, which Starlette/uvicorn surface to the
    # client as a bare HTTP 403 on the handshake rather than a WS close frame.
    client = make_client(membership=FakeMembership(member=False))
    with pytest.raises(WebSocketDisconnect):
        with client.websocket_connect(f"/ws?room={ROOM_ID}", headers={"x-room-token": token()}) as ws:
            ws.receive_text()


def test_accepts_valid_token_active_device_and_membership():
    client = make_client()
    with client.websocket_connect(f"/ws?room={ROOM_ID}", headers={"x-room-token": token()}) as ws:
        ws.send_text('{"app": "code"}')
        ws.close()
