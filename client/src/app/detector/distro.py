import platform
from pathlib import Path


def detect_distro() -> str | None:
    if platform.system().lower() != "linux":
        return None

    os_release = Path("/etc/os-release")
    if not os_release.exists():
        return None

    data = {}
    for line in os_release.read_text().splitlines():
        if "=" in line:
            key, _, value = line.partition("=")
            data[key.strip()] = value.strip().strip('"')

    distro_id = data.get("ID", "").lower()

    match distro_id:
        case "arch" | "manjaro" | "endeavouros":
            return "arch"
        case "ubuntu" | "debian" | "linuxmint":
            return "debian"
        case "fedora" | "rhel" | "centos":
            return "fedora"
        case _:
            return distro_id or None
