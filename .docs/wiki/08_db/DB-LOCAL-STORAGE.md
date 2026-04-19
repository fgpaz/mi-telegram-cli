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
      cursor.json
```

## 3. Reglas

- `profile.json` y `auth-state.json` representan metadata controlada por el CLI.
- `session.bin` es derivación física sensible.
- `lock.json` o equivalente debe reflejar exclusión operativa.
- `cursor.json` es opcional si la implementación decide persistir cursores.

## 4. Operaciones sensibles

- `profiles remove` debe purgar el árbol completo del perfil.
- `auth logout` debe invalidar `session.bin` y actualizar `auth-state.json`.

