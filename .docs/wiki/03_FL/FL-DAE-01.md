# FL-DAE-01 Coordinar comandos concurrentes por perfil

## Objetivo

Permitir que varios proyectos compartan perfiles ya logueados sin competir por `lock.json` ni duplicar sesiones por repo.

## Secuencia

1. El agente invoca un comando Telegram con `--profile`.
2. El CLI asegura daemon si el modo es `auto` o `required`.
3. El CLI crea un ticket FIFO bajo el perfil.
4. El comando espera su turno hasta `--queue-timeout`.
5. Al ejecutar, toma `lock.json`, opera y libera lock/ticket.
6. Si vence la espera, devuelve `QueueTimeout`.

## Errores visibles

- `QueueTimeout`
- `DaemonUnavailable`
- `DaemonLeaseDenied`
- `DaemonLeaseExpired`
