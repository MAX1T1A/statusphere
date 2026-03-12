# statusphere

Мониторинг системы в реальном времени для тебя и друзей. CPU, память, активные приложения — всё в терминале или браузере.

## Как работает

Клиент собирает метрики, шлёт на сервер по WebSocket, получает данные от других. Пока никого нет — спит. Кто-то подключился — просыпается.

## Запуск

```bash
go mod tidy

# терминал
go run ./cmd/client -ui tui

# браузер (localhost:8080)
go run ./cmd/client -ui web
```

Имя устройства задаётся через `DEVICE_NAME`:

```bash
DEVICE_NAME="мой ноут" go run ./cmd/client -ui tui
```

## Как добавить новую метрику

**1.** Провайдер — `internal/collector/linux/uptime.go`:

```go
package linux

import (
    "os"
    "strconv"
    "strings"
    "statusphere-client/internal/model"
)

func Uptime() func(model.Snapshot) {
    return func(snap model.Snapshot) {
        data, err := os.ReadFile("/proc/uptime")
        if err != nil {
            return
        }
        val, _ := strconv.ParseFloat(strings.Fields(string(data))[0], 64)
        snap["uptime_hours"] = val / 3600
    }
}
```

**2.** Регистрация в `cmd/client/main.go`:

```go
providers = append(providers, linuxc.Uptime())
```

**3.** Колонка — `internal/renderer/tui/col_uptime.go`:

```go
package tui

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

func ColUptime() Column {
    return Column{
        Header: "Uptime",
        Format: func(d map[string]any) string {
            if v, ok := d["uptime_hours"].(float64); ok {
                return fmt.Sprintf("%.1fh", v)
            }
            return "—"
        },
        Style: func(string) lipgloss.Style {
            return lipgloss.NewStyle().Align(lipgloss.Right).Padding(0, 1)
        },
    }
}
```

Добавь `ColUptime()` в список колонок в `tui.go`. Готово.

## Стек

- [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) — TUI
- [coder/websocket](https://github.com/coder/websocket) — WebSocket