# DB-DAEMON-AUDIT

## 1. Layout físico

```text
~/.mi-telegram-cli/
  daemon/state.json
  audit/events-YYYY-MM-DD.jsonl
  profiles/<profileId>/queue/<ticket>.ticket
  profiles/<profileId>/lease.json
```

## 2. Reglas

- `state.json` es local-only y contiene token de coordinación, no credenciales Telegram.
- Los tickets ordenan FIFO por perfil y deben borrarse al adquirir lock o vencer timeout.
- `lease.json` expira por tiempo y puede ser removido por el siguiente intento si ya venció.
- Los eventos JSONL son append-only diarios y redacted.

## 3. Campos prohibidos en auditoría

No guardar texto de mensajes, captions, códigos de auth, passwords, API hash, session blobs ni paths de archivos enviados.
