import asyncio
import signal

from app.transport import Transport
from app.transport.http import HTTPTransport
from app.watcher import SystemWatcher


async def main() -> None:
    transport: Transport = HTTPTransport(
        server_url="http://localhost:8000",
        token="my-room-token",
    )
    watcher = SystemWatcher(on_change=transport.send)

    await transport.start()
    await watcher.start()

    stop_event = asyncio.Event()
    loop = asyncio.get_event_loop()
    loop.add_signal_handler(signal.SIGINT, stop_event.set)
    loop.add_signal_handler(signal.SIGTERM, stop_event.set)

    async def on_event(data: dict) -> None:
        print("📡 получили снапшот:", data)

    listen_task = asyncio.create_task(transport.listen(on_event))

    await stop_event.wait()

    listen_task.cancel()
    await watcher.stop()
    await transport.stop()


asyncio.run(main())
