import asyncio
import json
import logging
import time

from app.core.auth.auth import verify_account_token
from app.services.account import AccountService
from app.services.membership import MembershipService
from app.services.providers import (
    provide_account_service_stub,
    provide_membership_service_stub,
    provide_room_manager_stub,
    provide_sampler_stub,
)
from app.services.room import RoomManager
from app.services.sampler import Sampler
from fastapi import APIRouter, Depends, WebSocket, WebSocketDisconnect

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
async def ws_endpoint(
    websocket: WebSocket,
    room: str = "",
    room_manager: RoomManager = Depends(provide_room_manager_stub),
    sampler: Sampler = Depends(provide_sampler_stub),
    accounts: AccountService = Depends(provide_account_service_stub),
    membership: MembershipService = Depends(provide_membership_service_stub),
) -> None:
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

    await websocket.accept()
    _active.add(websocket)
    logger.info("ws connected: account=%s device=%s room=%s", account_id, device_id, room)

    recv_task = None
    authz_task = None
    try:

        async def forward_to_client():
            async for data in room_manager.subscribe(room, device_id):
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

            await room_manager.publish(room, account_id, device_id, snapshot)
            await sampler.put(room, account_id, device_id, snapshot)

    except WebSocketDisconnect:
        logger.info("ws disconnected: device=%s", device_id)
    except Exception:
        logger.exception("ws error: device=%s", device_id)
    finally:
        _active.discard(websocket)
        for task in (recv_task, authz_task):
            if task:
                task.cancel()
