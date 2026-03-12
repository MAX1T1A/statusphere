import json
import logging

from app.services.providers import provide_room_manager_stub, provide_sampler_stub
from app.services.room import RoomManager
from app.services.sampler import Sampler
from fastapi import APIRouter, Depends, WebSocket, WebSocketDisconnect

logger = logging.getLogger(__name__)
router = APIRouter(tags=["ws"])


@router.websocket("/ws")
async def ws_endpoint(
    websocket: WebSocket,
    room_manager: RoomManager = Depends(provide_room_manager_stub),
    sampler: Sampler = Depends(provide_sampler_stub),
) -> None:
    token = websocket.headers.get("x-room-token")
    device_id = websocket.headers.get("x-device-id")

    if not token or not device_id:
        await websocket.close(code=1008, reason="missing headers")
        return

    await websocket.accept()
    logger.info("ws connected: device=%s room=%s", device_id, token[:8])

    recv_task = None
    try:
        import asyncio

        async def forward_to_client():
            async for data in room_manager.subscribe(token, device_id):
                await websocket.send_text(json.dumps(data))

        recv_task = asyncio.create_task(forward_to_client())

        while True:
            raw = await websocket.receive_text()
            snapshot = json.loads(raw)
            await room_manager.publish(token, device_id, snapshot)
            await sampler.put(token, device_id, snapshot)

    except WebSocketDisconnect:
        logger.info("ws disconnected: device=%s", device_id)
    except Exception:
        logger.exception("ws error: device=%s", device_id)
    finally:
        if recv_task:
            recv_task.cancel()
