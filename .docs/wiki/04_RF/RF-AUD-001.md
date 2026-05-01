# RF-AUD-001 Exportar y resumir auditoría JSONL redacted

## Requisito

El CLI debe registrar eventos JSONL diarios y permitir exportarlos o resumirlos localmente.

## Criterios

- Evento mínimo: `eventVersion`, `eventId`, timestamps, operación, perfil, `projectCwd`, pid, `daemonPid`, `queueMs`, `durationMs`, `ok`, `exitCode`, `errorCode`, `errorKind`, `peerQuery`.
- `audit export` soporta `--since`, `--profile`, `--operation` y `--errors-only`.
- `audit summary` agrupa por operación, perfil y proyecto con percentiles de cola/duración.
- Nunca persiste texto de mensajes, captions, códigos, passwords, API hash, session blobs ni paths de archivos enviados.
