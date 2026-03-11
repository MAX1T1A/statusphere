import json
import subprocess


def active_workspace() -> str | None:
    try:
        result = subprocess.run(
            ["hyprctl", "activeworkspace", "-j"],
            capture_output=True,
            text=True,
        )
        data = json.loads(result.stdout)
        return str(data.get("id"))
    except Exception:
        return None
