# RF-PRJ-001 - Vincular proyecto a perfil QA fijo

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-PRJ-001` |
| Titulo | Vincular proyecto a perfil QA fijo |
| Modulo | `PRJ` |
| Flow fuente | `FL-PRJ-01` |
| Actor | Operador tecnico, Agente |
| Trigger | `projects bind/list/show/current/remove` |
| Resultado observable | Binding persistido, visible y removible sin tocar Telegram |

## 2. Preconditions

- El root de proyecto es una ruta local normalizable.
- El perfil existe, salvo que `projects bind` reciba `--create-profile`.

## 3. Inputs

| Campo | Tipo | Req | Validacion |
| --- | --- | --- | --- |
| `root` | `path` | Sí para `bind/show/remove` | ruta local normalizada y limpia |
| `profileId` | `string` | Sí para `bind` | perfil existente o creable |
| `createProfile` | `bool` | No | default `false` |
| `displayName` | `string` | No | usado al crear metadata local |

## 4. Process Steps

1. Normalizar `root` a ruta absoluta limpia.
2. Validar que `profileId` exista.
3. Si falta y `--create-profile` está presente, crear `PerfilLocal` con `AuthorizationStatus=Unauthorized`.
4. Persistir o actualizar el binding en `~/.mi-telegram-cli/projects.json`.
5. Exponer `projects list/show/current/remove` sin incluir secretos ni sesión MTProto.

## 5. Typed Errors

| Code | Trigger |
| --- | --- |
| `ProfileNotFound` | `bind` sin perfil existente ni `--create-profile` |
| `ProjectBindingNotFound` | `show/remove` sin binding para ese root |
| `LocalStorageFailure` | error leyendo o escribiendo `projects.json` |

## 6. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-PRF-010` | bind con `--create-profile` crea metadata `Unauthorized` |
| `TP-PRF-011` | bind sin perfil existente falla con `ProfileNotFound` |
| `TP-PRF-012` | list/show/current/remove operan sobre `projects.json` |
