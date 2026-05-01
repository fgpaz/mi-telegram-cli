# CT-DAEMON-LOCAL

## 1. Comandos

- `daemon start`: inicia el coordinador local si no está activo.
- `daemon status`: informa `running`, `pid`, `host`, `port` y `startedAtUtc`.
- `daemon stop`: solicita cierre local.

## 2. Contrato runtime

El daemon escucha solo en `127.0.0.1`.
El token local se guarda en `~/.mi-telegram-cli/daemon/state.json`.
No hay endpoints remotos, admin UI ni pooling MTProto persistente.

## 3. Errores

- `DaemonUnavailable`
- `DaemonLeaseDenied`
- `DaemonLeaseExpired`
- `QueueTimeout`
