# DB-LOCAL-STORAGE

## 1. Objetivo

Documentar el layout físico mínimo para persistencia local por perfil.

## 2. Layout propuesto

```text
%USERPROFILE%\.mi-telegram-cli\
  profiles\
    <profileId>\
      profile.json
      auth-state.json
      session.bin
      lock.json
      lease.json
      queue\
        <ticket>.ticket
      cursor.json
  daemon\
    state.json
  audit\
    events-YYYY-MM-DD.jsonl
```

## 3. Reglas

- `profile.json` y `auth-state.json` representan metadata controlada por el CLI.
- `session.bin` es derivación física sensible.
- `lock.json` o equivalente debe reflejar exclusión operativa.
- `queue/<ticket>.ticket` materializa orden FIFO por perfil en modo daemon.
- `lease.json` protege `auth login` interactivo con TTL máximo de 10m.
- `cursor.json` es opcional si la implementación decide persistir cursores.
- `daemon/state.json` contiene solo host loopback, puerto, token local, pid y hora de inicio.
- `audit/events-YYYY-MM-DD.jsonl` contiene eventos operativos redacted.

## 4. Operaciones sensibles

- `profiles remove` debe purgar el árbol completo del perfil.
- `auth logout` debe invalidar `session.bin` y actualizar `auth-state.json`.
