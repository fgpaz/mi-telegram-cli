# FL-PRJ-01 - Vincular proyectos a perfiles QA fijos

## 1. Goal

Permitir que cada repo use un perfil QA fijo propio, resuelto automáticamente por `cwd`, para evitar que proyectos concurrentes compartan la misma sesión física.

## 2. Scope in/out

- In: bind/list/show/current/remove de bindings `projectRoot -> profileId`.
- In: `bind` solo con un perfil QA existente (`qa-*`), autorizado y con sesión local presente.
- Out: crear perfiles desde `bind`, login automático, migración o copia de `session.bin`.

## 3. Main sequence

```mermaid
sequenceDiagram
    participant User as Operador/Agente
    participant CLI as mi-telegram-cli
    participant Store as Storage perfiles
    participant Registry as projects.json

    User->>CLI: projects bind --root R --profile P
    CLI->>Store: validar existencia, QA y autorización local de P
    alt perfil válido
        CLI->>Registry: persistir R -> P
        User->>CLI: messages send sin --profile desde R/subdir
        CLI->>Registry: resolver prefijo mas largo por cwd
        CLI->>Store: cargar perfil P
        CLI-->>User: salida con profile=P
    else perfil ausente/no autorizado/no QA
        CLI-->>User: error tipado; no cambia projects.json
    end
```

## 4. Error path

- Perfil ausente durante `bind`: `ProfileNotFound`.
- Perfil no QA, no autorizado o sin sesión local: `ProfileNotEligibleForBinding`.
- Perfil protegido: `ProfileProtected`.
- Binding ausente durante `show/remove`: `ProjectBindingNotFound`.
- Binding efectivo apunta a perfil eliminado: `ProjectProfileMissing`.

## 5. RF references

- `RF-PRJ-001`
- `RF-PRJ-002`
