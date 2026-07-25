# Statusphere

Видишь, кто что слушает, в каком приложении сидит и сколько онлайн — у всех в комнате, прямо в терминале.

## Экраны

_Скриншоты (кладутся в `docs/`):_

### Комната
![Комната](screenshot.png)

### Музыка
![Музыка](docs/music.png)

### Экранное время
![Экранное время](docs/screen.png)

### Чат комнаты
![Чат](docs/chat.png)

### Меню действий
![Меню](docs/menu.png)

## Установка

```bash
curl -sSL https://raw.githubusercontent.com/MAX1T1A/statusphere/master/install.sh | bash
```

Ставит бинарник в `~/.local/bin/` и добавляет алиас `sstatus`. После установки перезапусти шелл (новый терминал или `source ~/.bashrc` / `source ~/.zshrc`).

Нужен Linux.

## Первый запуск

Создать аккаунт (своя комната, ты — владелец):

```bash
sstatus --register https://ss.ug3n.com
```

Запустить:

```bash
sstatus
```

## Подключиться к друзьям

Позвать к себе в комнату:

```bash
sstatus --invite          # покажет инвайт (в нём зашит твой сервер, действует 1 час)
```

Войти по инвайту друга — аккаунт заведётся сам на его сервере, отдельная регистрация не нужна:

```bash
sstatus --join <инвайт>
sstatus                   # и смотришь комнату
```

## Второе устройство

```bash
sstatus --new-device                              # на старом устройстве → печатает код
sstatus --link https://ss.ug3n.com --code <код>   # на новом устройстве
```

Новое устройство попадает в ту же комнату.

## Управление

```bash
sstatus --set-name "Имя"        # как тебя видят в комнате
sstatus --devices               # список твоих устройств
sstatus --revoke <device_id>    # отозвать устройство
sstatus --members               # кто в комнате
sstatus --kick <account_id>     # выгнать участника (только владелец)
```

Потерял все устройства — восстановись по сохранённым `account_id` и `account_secret`:

```bash
sstatus --recover https://ss.ug3n.com --account <account_id> --secret <account_secret>
```

## В интерфейсе

- `↑ ↓` — выбрать человека
- `Enter` — меню действий: музыка, экранное время, синхронизация трека, чат, переименование
- `q` — выход · `Esc` — закрыть окно
