import asyncio
import json
import logging
import time

from app.modules.chats.application.commands.send_message import SendMessage
from app.modules.chats.domain.message import MAX_TEXT
from app.modules.presence.application.commands.ingest_snapshot import IngestPresenceSnapshot
from app.platform.security import verify_account_token
from app.shared_kernel.actor import Actor
from fastapi import APIRouter, WebSocket, WebSocketDisconnect

logger = logging.getLogger(__name__)
router = APIRouter(tags=["ws"])

MAX_MESSAGE_SIZE = 16 * 1024
MIN_MESSAGE_INTERVAL = 0.5
AUTHZ_RECHECK = 10.0

_active: set[WebSocket] = set()


async def close_all() -> None:
    for ws in list(_active):
        try:
            await ws.close(code=1001, reason="server shutting down")
        except Exception:
            pass
    _active.clear()


@router.websocket("/ws")
async def ws_endpoint(websocket: WebSocket, room: str = "") -> None:
    container = websocket.app.state.container
    bus = container.bus
    hub = container.hub
    accounts = container.accounts
    membership = container.membership

    identity = verify_account_token(websocket.headers.get("x-room-token", ""))
    if identity is None:
        await websocket.close(code=1008, reason="invalid token")
        return

    account_id, device_id = identity

    if not await accounts.is_device_active(account_id, device_id):
        await websocket.close(code=1008, reason="device revoked")
        return

    if not room or not await membership.is_member(room, account_id):
        await websocket.close(code=1008, reason="not a room member")
        return

    account_name = await accounts.name_of(account_id)
    actor = Actor(account_id=account_id, device_id=device_id)

    await websocket.accept()
    _active.add(websocket)
    logger.info("ws connected: account=%s device=%s room=%s", account_id, device_id, room)

    recv_task = None
    authz_task = None
    try:

        async def forward_to_client():
            async for data in hub.subscribe(room, device_id, account_id):
                await websocket.send_text(json.dumps(data))

        async def watch_authz():
            while True:
                await asyncio.sleep(AUTHZ_RECHECK)
                active = await accounts.is_device_active(account_id, device_id)
                member = await membership.is_member(room, account_id)
                if not active or not member:
                    logger.info("ws access revoked: account=%s device=%s", account_id, device_id)
                    await websocket.close(code=1008, reason="access revoked")
                    return

        recv_task = asyncio.create_task(forward_to_client())
        authz_task = asyncio.create_task(watch_authz())

        last_msg_at = 0.0
        while True:
            raw = await websocket.receive_text()
            if len(raw) > MAX_MESSAGE_SIZE:
                await websocket.close(code=1009, reason="message too large")
                return

            now = time.monotonic()
            if now - last_msg_at < MIN_MESSAGE_INTERVAL:
                continue
            last_msg_at = now

            try:
                snapshot = json.loads(raw)
            except json.JSONDecodeError:
                continue

            if not isinstance(snapshot, dict):
                continue

            if snapshot.get("type") == "msg":
                text = (snapshot.get("text") or "").strip()[:MAX_TEXT]
                if not text:
                    continue
                to_account = (snapshot.get("to") or "").strip()
                await bus.dispatch(
                    SendMessage(
                        actor=actor,
                        room_token=room,
                        to_account=to_account,
                        text=text,
                        from_name=account_name,
                    )
                )
                continue

            await bus.dispatch(
                IngestPresenceSnapshot(actor=actor, room=room, account_name=account_name, snapshot=snapshot)
            )

    except WebSocketDisconnect:
        logger.info("ws disconnected: device=%s", device_id)
    except Exception:
        logger.exception("ws error: device=%s", device_id)
    finally:
        _active.discard(websocket)
        for task in (recv_task, authz_task):
            if task:
                task.cancel()
