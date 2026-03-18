# Statusphere

Утилита для обмена присутствием между друзьями в реальном времени. Видишь что слушают, в каком приложении сидят, сколько онлайн. Работает как TUI прямо в терминале.

## Как это выглядит

![screenshot](screenshot.png)

Каждый клиент собирает информацию о системе, отправляет на общий сервер по WebSocket и отображает всех подключённых друзей в виде карточек.

## Установка

```bash
curl -sSL https://raw.githubusercontent.com/MAX1T1A/statusphere/master/install.sh | bash
```

Скрипт клонит репозиторий, собирает бинарник, ставит в `~/.local/bin/` и добавляет алиас `sstatus` в конфиг шелла. После установки перезапусти шелл или выполни `source ~/.bashrc` / `source ~/.zshrc`.

Требования: `curl` или `wget`.

## Возможности

**Присутствие и системная информация**
- Статус онлайн/idle/офлайн с цветным индикатором
- Аптайм системы
- Активное приложение и заголовок окна (Hyprland)

**Spotify**
- Текущий трек: исполнитель, название, альбом + пиксельная обложка (half-block рендеринг)
- Статистика прослушивания за неделю с графиком по дням
- Синхронизация — нажми `s` чтобы послушать тот же трек что и друг (через D-Bus)

**Экранное время**
- Бары использования приложений за день (данные с сервера `/stats/summary`)
- Градиентная цветовая палитра (purple → cyan)
- Кэш per-device с фоновым обновлением

**Сообщения (Nudge)**
- Нажми `n` чтобы отправить сообщение всем в комнате
- Сообщения отображаются внутри карточки отправителя рядом с блоком Spotify
- Десктопные уведомления через `notify-send`
- История чата с временными метками, свои сообщения выделены серым

**Управление устройством**
- Переименование через `d` — сохраняется в `~/.config/statusphere/device_name`

## Архитектура

```
collector/       Провайдеры данных (подключаются в зависимости от ОС/DE/дистрибутива)
  linux/         Linux: uptime, spotify
    arch/        Arch Linux: количество пакетов
    hyprland/    Hyprland: активное окно, воркспейс
    spotify/     Spotify через D-Bus MPRIS
detector/        Автоопределение ОС, дистрибутива, DE/WM, терминала
transport/       WebSocket клиент с автореконнектом, идентификация устройства
watcher/         Поллит коллекторы, сравнивает снапшоты, шлёт при изменении
                 InjectOnce() — одноразовые поля в следующий снапшот (nudge)
feed/            Агрегирует входящие данные устройств из WebSocket
stats/           Интерфейс Fetcher + per-device Cache (async, stale-while-revalidate)
  spotify.go     Статистика прослушивания Spotify
  summary.go     Статистика использования приложений
renderer/
  tui/           BubbleTea TUI с карточным лейаутом
    block_header    Статус, имя, аптайм
    block_spotify   Обложка, текущий трек, недельный график
    block_app       Активное приложение, бары экранного времени
    block_nudge     История сообщений per-device
  noop/          Headless режим (без UI, только сбор и отправка)
notifier/        Десктопные уведомления через notify-send
auth/            Регистрация и хранение токена авторизации
models/          Тип Snapshot (map[string]any)
```

### Добавить новый провайдер

Создай файл в `collector/linux/`, который возвращает `func(models.Snapshot)`:

```go
package linux

import "statusphere-client/internal/models"

func MyProvider() func(models.Snapshot) {
    return func(snap models.Snapshot) {
        snap["my_key"] = "my_value"
    }
}
```

Зарегистрируй его в `cmd/client/main.go` внутри `buildProviders()`.

### Добавить новую статистику

Создай файл в `stats/`, реализующий интерфейс `Fetcher`:

```go
package stats

import "net/url"

type MyData struct { /* ... */ }

type myFetcher struct{}

func (f myFetcher) Path() string                    { return "/stats/my-endpoint" }
func (f myFetcher) Query(deviceID string) url.Values { /* ... */ }
func (f myFetcher) New() any                         { return &MyData{} }

func NewMyCache(serverURL, token string) *Cache {
    return NewCache(serverURL, token, myFetcher{})
}
```

## Требования

- Go 1.25+
- Linux (провайдеры используют `/proc`, `/sys`, D-Bus)
- Hyprland (для активного окна/воркспейса — опционально)
- Десктопный Spotify (для текущего трека / синхронизации — опционально)
- Запущенный сервер Statusphere

## Настройка сервера

Сервер требует переменную окружения `AUTH_SECRET` — произвольная строка, используется для подписи токенов авторизации (HMAC-SHA256). Без неё сервер не запустится.

```bash
export AUTH_SECRET="$(openssl rand -hex 32)"
export POSTGRES_DB_HOST=localhost
export POSTGRES_DB_PORT=5432
export POSTGRES_DB_LOGIN=postgres
export POSTGRES_DB_PASSWORD=your-password
export POSTGRES_DB_NAME=statusphere
```

Переменные для PostgreSQL **обязательны** — дефолтных значений нет, сервер упадёт с ошибкой если что-то не задано.

### Docker

```bash
# Задай переменные в .env файле
echo 'AUTH_SECRET=your-secret-here' >> .env
echo 'POSTGRES_DB_LOGIN=postgres' >> .env
echo 'POSTGRES_DB_PASSWORD=your-password' >> .env
# ...

docker compose up -d
```

## Регистрация и запуск клиента

```bash
cd client
go build -o sstatus ./cmd/client
```

При первом запуске клиент должен зарегистрироваться на сервере. Регистрация создаёт подписанный токен, который привязывает устройство к комнате.

**Создать новую комнату:**

```bash
sstatus --register https://your-server.com
```

Сервер сгенерирует `room_id` (128-бит random hex) и вернёт токен. Скопируй `room_id` из вывода и передай друзьям.

**Присоединиться к существующей комнате:**

```bash
sstatus --register https://your-server.com --room <room_id>
```

Конфигурация сохраняется в `~/.config/statusphere/config.json` (права `0600`):

```json
{
  "server_url": "https://your-server.com",
  "token": "room_id:device_id:hmac_signature"
}
```

После регистрации просто запускай:

```bash
# TUI режим (по умолчанию)
sstatus

# Headless режим (только сбор и отправка, без UI)
sstatus --ui headless
```

## Безопасность

Все подключения (WebSocket и HTTP) авторизуются через HMAC-подписанный токен в заголовке `X-Room-Token`. Токен содержит `room_id`, `device_id` и подпись — подменить комнату или устройство невозможно без знания серверного секрета.

- **Регистрация** — `POST /auth/register`, rate limit 5 запросов/мин на IP
- **Stats API** — `GET /stats/summary`, `GET /stats/spotify`, rate limit 30 запросов/мин на IP, `room_id` берётся из токена
- **WebSocket** — максимальный размер сообщения 16KB, rate limit 2 msg/sec на соединение
- **Graceful shutdown** — при остановке сервера все WS-клиенты получают clean close (код 1001)

## Горячие клавиши

| Клавиша | Действие |
|---------|----------|
| `n` | Отправить сообщение всем в комнате |
| `d` | Переименовать устройство |
| `s` | Синхронизировать трек Spotify с другом |
| `q` | Выход |
| `Esc` | Отменить ввод |

## Конфигурация

Вся конфигурация клиента хранится в `~/.config/statusphere/`:

- `config.json` — URL сервера, токен авторизации (создаётся при `--register`)
- `device_name` — отображаемое имя (задаётся через `d`)

Логи пишутся в `~/.cache/statusphere/statusphere.log`.