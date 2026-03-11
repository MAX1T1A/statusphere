# Statusphere

Share your device status with friends in real time. No polling — only push when something changes.

## How it works

Each device runs a **client** that watches the system state. When something changes (active window, workspace, CPU, RAM) — it pushes a snapshot to the **server**. The server broadcasts it to everyone in the same room via SSE.

```
[Device A]──POST /status──▶ [Server] ──SSE /feed──▶ [Device B]
[Device B]──POST /status──▶ [Server] ──SSE /feed──▶ [Device A]
```

No unnecessary traffic — the client only sends when state actually changes.

## Project structure

```
statusphere/
├── client/       # runs on each device
└── server/       # self-hosted, one instance for a group of friends
```

## Server

### Run with Docker

```bash
cd server
docker compose up -d
```

### Run locally

```bash
cd server
uv sync
PYTHONPATH=src uv run uvicorn app:app --host 0.0.0.0 --port 8000
```

### API

| Method | Endpoint | Headers | Description |
|--------|----------|---------|-------------|
| `POST` | `/status` | `X-Room-Token`, `X-Device-Id` | Push a snapshot |
| `GET` | `/feed` | `X-Room-Token`, `X-Device-Id` | Subscribe to room updates via SSE |

## Client

### Requirements

- Python 3.12+
- [uv](https://github.com/astral-sh/uv)

### Run

```bash
cd client
uv sync
PYTHONPATH=src uv run python -m app
```

### Configuration

Set via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_URL` | `http://localhost:8000` | Server address |
| `ROOM_TOKEN` | `my-room-token` | Shared token for a group of friends |
| `DEVICE_ID` | MAC-based UUID | Unique device identifier |

### Supported platforms

| OS | Distro | DE / WM | Status |
|----|--------|---------|--------|
| Linux | any | any | ✅ CPU, RAM, load avg |
| Linux | any | Hyprland | ✅ + active window, app, workspace |
| Windows | — | — | 🚧 planned |
| macOS | — | — | 🚧 planned |
| Android | — | — | 🚧 planned |

## Collected data

```json
{
  "device_id": "...",
  "cpu_percent": 12.4,
  "memory_used_mb": 4200.0,
  "memory_total_mb": 16384.0,
  "load_avg_1m": 0.85,
  "active_workspace": "3",
  "active_window": "statusphere - Visual Studio Code",
  "active_app": "code"
}
```

Fields are `null` if not supported on the current platform.

## Adding a new collector

1. Create a file inside the appropriate platform folder:

```
client/src/app/collector/linux/my_feature.py
```

2. Write a function that returns the value:

```python
def my_value() -> str | None:
    try:
        ...
    except Exception:
        return None
```

3. Add the field to `Snapshot`:

```python
@dataclass
class Snapshot:
    ...
    my_value: str | None = None
```

4. Call it in the corresponding collector:

```python
class LinuxCollector:
    def collect(self, snapshot: Snapshot) -> None:
        ...
        snapshot.my_value = my_value()
```

That's it. The server doesn't need to be touched — it's data-agnostic.

## Rooms

A room is a group of friends sharing statuses. Everyone with the same token is in the same room.

1. Pick a token (any string)
2. Share it with friends
3. Set `ROOM_TOKEN=your-token` on each device

Private channels (per-user subscriptions) are planned.

## Self-hosting

The server is a single stateless FastAPI app with in-memory storage. Deploy it anywhere Docker runs — a VPS, a home server, Dokploy, Coolify.

```bash
docker compose up -d
```

## License

MIT