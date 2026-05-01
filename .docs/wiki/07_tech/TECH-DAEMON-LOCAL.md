# TECH-DAEMON-LOCAL

## 1. Topología

El daemon es un proceso local de usuario, headless y loopback-only.
Su estado vive en `~/.mi-telegram-cli/daemon/state.json` con host `127.0.0.1`, puerto, token local, pid y `startedAtUtc`.
No hay admin UI, endpoint remoto ni pooling persistente MTProto en v1.

## 2. Modos

- `MI_TELEGRAM_CLI_DAEMON=auto`: modo por defecto del binario distribuido; asegura daemon y usa cola.
- `MI_TELEGRAM_CLI_DAEMON=off`: ejecuta modo directo y conserva `ProfileLocked`.
- `MI_TELEGRAM_CLI_DAEMON=required`: falla con `DaemonUnavailable` si el daemon no puede usarse.

## 3. Cola y lease

La cola es FIFO por perfil mediante tickets en storage local.
Perfiles distintos pueden ejecutar en paralelo.
El timeout default es 120s y puede configurarse con `MI_TELEGRAM_CLI_QUEUE_TIMEOUT_SECONDS` o `--queue-timeout`.
`auth login` toma una lease externa con TTL = timeout de login + 30s, máximo 10m.

## 4. Seguridad

El token local no es credencial Telegram.
Los eventos de auditoría y el daemon nunca guardan texto de mensajes, captions, códigos, passwords, API hash, session blobs ni paths de archivos enviados.
