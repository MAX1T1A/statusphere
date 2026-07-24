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
- Аптайм системы, загрузка CPU, память, load average
- Количество пакетов (Arch)
- Активное приложение, заголовок окна и воркспейс (Hyprland)

**Кастомные поля**
- Свои поля из произвольных shell-команд (`~/.config/statusphere/custom.json`)
- Отображаются отдельным блоком в карточке
- Синхронизация полей с другом через `s`

> ⚠️ Кастомные поля выполняют произвольные команды через `sh -c`. Не добавляй в `custom.json` команды из недоверенных источников.

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

Клиент (`client/internal/`):

```
app/             Жизненный цикл: сборка провайдеров, оркестрация горутин, реализация tui.Controller
presence/        Типизированный Snapshot: реестр ключей (keys.go) + безопасные аксессоры (String/Int/Float/Strings)
config/          Единые пути ~/.config/statusphere, ~/.cache/statusphere
collector/       Реестр провайдеров: Provider{Name, Collect(ctx, snap) error} + Descriptor{Applies(detector.Context)}
                 Провайдеры самрегистрируются в init(); Collect() гоняет каждого с пер-провайдерным таймаутом
  linux/         uptime, cpu, memory, load, music (playerctl)
    arch/        Количество пакетов (pacman)
    hyprland/    Активное окно + воркспейс (hyprctl)
    spotify/     Текущий трек через D-Bus MPRIS
  custom/        Manager: поля из custom.json как провайдеры (кэш per-field, TTL)
detector/        Автоопределение ОС, дистрибутива, DE/WM, терминала
transport/       WebSocket клиент с автореконнектом; Send клонирует снапшот и штампует device_id/name
watcher/         Поллит коллекторы; шлёт при значимом изменении (волатильные метрики игнорируются) + heartbeat
                 InjectOnce() — одноразовые поля + мгновенный триггер отправки (nudge)
feed/            Агрегирует входящие устройства; отсекает «призраков» по TTL
media/           Управление Spotify через MPRIS (синхронизация трека)
stats/           Интерфейс Fetcher + per-device Cache (async, stale-while-revalidate)
renderer/
  tui/           BubbleTea TUI; Controller + Options вместо длинного конструктора
    block_header    Статус, имя, аптайм, системные метрики
    block_spotify   Обложка, текущий трек, недельный график
    block_app       Активное приложение, воркспейс, бары экранного времени
    block_nudge     История сообщений per-device
    block_custom    Кастомные поля
  noop/          Headless режим (без UI, только сбор и отправка)
notifier/        Десктопные уведомления через notify-send
auth/            Регистрация и хранение токена авторизации
```

Сервер (`server/src/app/`):

```
api/routes/      ws (broadcast + membership/revocation checks), stats, accounts, devices, rooms
services/account/    register, device link/revoke, recover
services/membership/ invite, join, members, kick (owner-only)
services/room/       Live-комнаты и подписчики, fan-out по WebSocket
services/sampler/    Буфер снапшотов + периодический flush в БД (re-queue при сбое)
services/snapshot/   Агрегация статистики (decrypt + aggregate в Python)
repositories/    SQL: snapshots (TimescaleDB, зашифрованный BYTEA), accounts, devices, room_members
core/config/     Settings — единый источник env (+ presence_key)
core/auth/       HMAC токены v2, коды приглашений/привязки с TTL, верификатор секрета
core/crypto/     AES-256-GCM шифрование payload'а
core/ratelimit/  Rate limiter per-IP
```

### Добавить новый провайдер

Создай файл в `collector/linux/` (или в новом пакете под платформу) — провайдер сам регистрируется в `init()`:

```go
package linux

import (
    "context"

    "statusphere-client/internal/collector"
    "statusphere-client/internal/presence"
)

func init() {
    collector.Register(collector.Descriptor{
        Provider: collector.Provider{Name: "my-provider", Collect: myProvider},
        Applies:  collector.OnOS("linux"),
    })
}

func myProvider(ctx context.Context, snap presence.Snapshot) error {
    snap.Set("my_key", "my_value")
    return nil
}
```

Ключ добавь в `presence/keys.go`. Никакого switch в wiring править не нужно — реестр подхватит провайдер автоматически (нужен blank-импорт пакета в `app/app.go`). `Applies` определяет, где провайдер активен: `OnOS`, `OnDistro`, `OnDEWM`, `When(...)`.

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
export PRESENCE_KEY="$(openssl rand -base64 32)"   # ключ шифрования данных в БД (base64 32 байт)
export POSTGRES_DB_HOST=localhost
export POSTGRES_DB_PORT=5432
export POSTGRES_DB_LOGIN=postgres
export POSTGRES_DB_PASSWORD=your-password
export POSTGRES_DB_NAME=statusphere
```

`PRESENCE_KEY` **обязателен** — им шифруется payload снапшотов в БД (AES-256-GCM). Потеря ключа = данные не расшифровать (для эфемерного presence это приемлемо, retention 7 дней). Не меняй ключ на работающей базе — старые записи станут нечитаемыми.

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

## Аккаунты, устройства, комнаты

Модель авторизации — **аккаунты** (без паролей). Аккаунту принадлежат **устройства**, комнаты состоят из аккаунтов. Токен устройства: `v2:account_id:device_id:HMAC` в заголовке `X-Room-Token`.

```bash
cd client
go build -o sstatus ./cmd/client
```

**Создать аккаунт** (генерит секрет клиента, создаёт личную комнату, ты — владелец):

```bash
sstatus --register https://your-server.com
```

**Позвать друзей / войти в общую комнату:**

```bash
sstatus --invite            # у себя: код-приглашение (TTL 1 ч)
sstatus --join <code>       # у друга: войти в твою комнату
```

**Добавить второе устройство** (без пароля, по коду с авторизованного):

```bash
sstatus --new-device                       # на старом устройстве -> печатает код
sstatus --link https://your-server.com --code <code>   # на новом устройстве
```

Новое устройство попадает в ту же комнату, где сейчас основное.

**Управление:**

```bash
sstatus --devices           # список устройств аккаунта
sstatus --revoke <device_id># отозвать устройство (рвёт и живую сессию)
sstatus --members           # кто в твоей комнате
sstatus --kick <account_id> # выгнать участника (только владелец)
```

**Восстановление** (если потерял все устройства — нужны сохранённые `account_id` и `account_secret`):

```bash
sstatus --recover https://your-server.com --account <account_id> --secret <account_secret>
```

Конфигурация — `~/.config/statusphere/config.json` (права `0600`): `server_url`, `account_secret`, `account_id`, `device_id`, `token`, `room_id`.

Запуск:

```bash
sstatus                # TUI (по умолчанию)
sstatus --ui headless  # только сбор и отправка, без UI
```

## Безопасность

Все подключения авторизуются v2-токеном устройства (`X-Room-Token`), подписанным HMAC-SHA256 серверным `AUTH_SECRET`.

- **Аккаунт** идентифицируется секретом клиента; сервер хранит только его HMAC-**верификатор**, не сам секрет — дамп БД учётки не выдаёт.
- **Устройства и членство** — WS-подключение (`/ws?room=<room_id>`) проверяет: валидность токена → устройство не отозвано → аккаунт состоит в комнате. Проверки повторяются на живой сессии (watchdog ~10с), поэтому **отзыв устройства и кик участника рвут активное соединение**, а не только новые.
- **Шифрование данных в БД (at-rest)** — payload снапшотов (`data`) шифруется AES-256-GCM; ключ `PRESENCE_KEY` живёт только в приложении и в БД не попадает. Дамп без ключа = бесполезный шифртекст. Это не end-to-end: серверу в рантайме доверяем.
- **Авторитетные `account_id`/`device_id`** — сервер проставляет их из проверенного токена при вещании и записи; значения из payload игнорируются.
- **Приглашения** — код в комнату с TTL 1 ч; **link-код** привязан к выдавшему устройству и протухает при его отзыве.
- **Rate limits** — регистрация/восстановление 5/мин на IP, stats 30/мин на IP, WS 2 msg/sec на соединение, max сообщение 16KB.
- **Graceful shutdown** — при остановке сервера все WS-клиенты получают clean close (1001).

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
- `custom.json` — кастомные поля (команды)

Логи пишутся в `~/.cache/statusphere/statusphere.log`.

### Кастомные поля

`~/.config/statusphere/custom.json` — порядок ключей сохраняется как в файле:

```json
{
  "weather": { "cmd": "curl -s 'wttr.in/Almaty?format=%C+%t'", "repeat_seconds": 900 },
  "git": { "cmd": "git -C ~/proj rev-parse --abbrev-ref HEAD", "repeat_seconds": 30 }
}
```

`repeat_seconds` — TTL кэша значения (команда не запускается чаще).

**Вывод поля — это ровно то, что печатает твоя команда.** Хочешь текстом — `wttr.in` формат `%C` («Partly cloudy»); хочешь значок-эмодзи — `%c`. Statusphere ничего не добавляет и не убирает из вывода: сам UI минималистичный и без эмодзи, но контент кастом-полей полностью твой.

### Текст в UI — откуда что

Три источника текста, чтобы понимать, что можно стилизовать/контролировать:

- **Chrome (наш)** — метки `music/app/chat/cfg`, разделители, подсказки, рамки, `you:`/имена, статусы `●◐○`. Всегда **серый**, текст, без эмодзи. Живёт в `renderer/tui`.
- **Значения (наши, из чисел провайдеров)** — аптайм, `cpu%`, память, длительности, счётчики, спарклайны. **Яркие/акцент**. Форматируются в коллекторах/блоках.
- **Сырьё источника (не наше)** — заголовок окна (hyprctl), артист/трек (MPRIS), вывод custom-команд. Проходит **дословно**, мы только красим контейнер.

Правило: **серое = наше, яркое = данные**; сырьё источника — «чужой текст как есть».

## Разработка

Клиент:

```bash
cd client
go build ./...
go test ./...
go vet ./...
```

Сервер:

```bash
cd server
uv sync
uv run pytest
```