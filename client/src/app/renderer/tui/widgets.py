from rich.text import Text


def format_cpu(value: float | None) -> Text:
    if value is None:
        return Text("—", style="dim")
    style = "green" if value < 50 else "yellow" if value < 80 else "bold red"
    return Text(f"{value:.1f}%", style=style)


def format_memory(used: float | None, total: float | None) -> str:
    if used is None or total is None:
        return "—"
    pct = (used / total) * 100 if total > 0 else 0
    return f"{used:.0f}/{total:.0f} MB ({pct:.0f}%)"


def format_window(app: str | None, title: str | None) -> Text:
    if not app and not title:
        return Text("—", style="dim")
    if not app:
        return Text(title, style="dim")
    if not title:
        return Text(app, style="bold")
    result = Text()
    result.append(app, style="bold")
    result.append(f" — {title}", style="dim")
    return result
