import json
import subprocess


def _hyprctl_activewindow() -> dict:
    result = subprocess.run(
        ["hyprctl", "activewindow", "-j"],
        capture_output=True,
        text=True,
    )
    return json.loads(result.stdout)


def active_window() -> str | None:
    try:
        return _hyprctl_activewindow().get("title")
    except Exception:
        return None


def active_app() -> str | None:
    try:
        return _hyprctl_activewindow().get("class")
    except Exception:
        return None
