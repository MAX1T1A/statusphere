import platform


def detect_os() -> str:
    system = platform.system().lower()

    match system:
        case "linux":
            return "linux"
        case "windows":
            return "windows"
        case "darwin":
            return "macos"
        case _:
            return "unknown"
