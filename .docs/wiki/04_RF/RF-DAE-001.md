# RF-DAE-001 Coordinar daemon local y cola FIFO por perfil

## Requisito

El CLI debe coordinar comandos Telegram concurrentes mediante daemon local de usuario y cola FIFO por perfil.

## Criterios

- Auto-start para `auth status/logout`, `me`, `dialogs *` y `messages *`.
- `MI_TELEGRAM_CLI_DAEMON=off` conserva modo directo.
- `MI_TELEGRAM_CLI_DAEMON=required` falla con `DaemonUnavailable` si no hay daemon usable.
- `--queue-timeout` y `MI_TELEGRAM_CLI_QUEUE_TIMEOUT_SECONDS` controlan la espera.
- Timeout antes de ejecutar devuelve `QueueTimeout`, no `ProfileLocked`.
- `auth login` usa lease externa con TTL timeout + 30s, cap 10m.
