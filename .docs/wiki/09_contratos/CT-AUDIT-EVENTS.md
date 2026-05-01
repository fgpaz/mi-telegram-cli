# CT-AUDIT-EVENTS

## 1. Evento JSONL v1

Campos mínimos:

- `eventVersion`
- `eventId`
- `startedAtUtc`
- `completedAtUtc`
- `operation`
- `profile`
- `projectCwd`
- `pid`
- `daemonPid`
- `queueMs`
- `durationMs`
- `ok`
- `exitCode`
- `errorCode`
- `errorKind`
- `peerQuery`

## 2. Comandos

- `audit export`: emite eventos filtrados.
- `audit summary`: resume conteos, errores y percentiles por operación, perfil y proyecto.

## 3. Redacción obligatoria

Nunca guardar texto de mensajes, captions, códigos, passwords, API hash, session blobs ni paths de archivos enviados.
