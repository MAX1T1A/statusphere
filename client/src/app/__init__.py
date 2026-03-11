import asyncio

from app.renderer import FeedRenderer
from app.transport import Transport
from app.transport.http import HTTPTransport
from app.watcher import SystemWatcher


async def main() -> None:
    transport: Transport = HTTPTransport(
        server_url="https://max1t1a-statusphere-znycvf-8e6cc4-193-181-208-72.traefik.me",
        token="my-room-token",
    )
    watcher = SystemWatcher(on_change=transport.send)

    async def on_ready():
        await transport.start()
        await watcher.start()
        asyncio.create_task(transport.listen(renderer.update))

    renderer = FeedRenderer(on_ready=on_ready)
    await renderer.run()

    await watcher.stop()
    await transport.stop()


asyncio.run(main())
