# FL-PRJ-01 - Vincular proyectos a perfiles QA fijos

## 1. Goal

Permitir que cada repo use un perfil QA fijo propio, resuelto automáticamente por `cwd`, para evitar que proyectos concurrentes compartan la misma sesión física.

## 2. Scope in/out

- In: bind/list/show/current/remove de bindings `projectRoot -> profileId`.
- In: creación opcional de metadata local no autorizada con `--create-profile`.
- Out: login automático, migración o copia de `session.bin`.

## 3. Main sequence

```mermaid
sequenceDiagram
    participant User as Operador/Agente
    participant CLI as mi-telegram-cli
    participant Registry as projects.json
    participant Store as Storage perfiles

    User->>CLI: projects bind --root R --profile P --create-profile
    CLI->>Store: asegurar PerfilLocal P si se solicita
    CLI->>Registry: persistir R -> P
    User->>CLI: messages send sin --profile desde R/subdir
    CLI->>Registry: resolver prefijo mas largo por cwd
    CLI->>Store: cargar perfil P
    CLI-->>User: salida con profile=P
```

## 4. Error path

- Perfil ausente durante `bind` sin `--create-profile`: `ProfileNotFound`.
- Binding ausente durante `show/remove`: `ProjectBindingNotFound`.
- Binding efectivo apunta a perfil eliminado: `ProjectProfileMissing`.

## 5. RF references

- `RF-PRJ-001`
- `RF-PRJ-002`
