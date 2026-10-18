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
- El perfil ya existe, su ID es `qa-*`, no está protegido, su estado local es `Authorized` y su sesión existe.
- `projects bind` no crea perfiles, inicia sesión ni accede a Telegram.

## 3. Inputs

| Campo | Tipo | Req | Validacion |
| --- | --- | --- | --- |
| `root` | `path` | Sí para `bind/show/remove` | ruta local normalizada y limpia |
| `profileId` | `string` | Sí para `bind` | perfil QA existente y autorizado; no admite creación desde bind |

## 4. Process Steps

1. Rechazar flags de creación de perfil como entrada inválida.
2. Validar que el perfil exista; exigir prefijo `qa-`, estado `Authorized`, sesión local presente y ausencia de protección.
3. Si cualquier validación falla, devolver error tipado antes de escribir.
4. Normalizar `root` a ruta absoluta limpia.
5. Persistir o actualizar el binding en `~/.mi-telegram-cli/projects.json`.
6. Exponer `projects list/show/current/remove` sin incluir secretos ni sesión MTProto.

## 5. Typed Errors

| Code | Trigger |
| --- | --- |
| `ProfileNotFound` | `bind` apunta a un perfil inexistente |
| `ProfileNotEligibleForBinding` | perfil existente no es QA, no está autorizado o no tiene sesión local |
| `ProfileProtected` | perfil protegido contra automatización |
| `InvalidInput` | flag de creación de perfil no admitido |
| `ProjectBindingNotFound` | `show/remove` sin binding para ese root |
| `LocalStorageFailure` | error leyendo o escribiendo `projects.json` |

## 6. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-PRF-010` | bind con flag de creación rechaza sin crear perfil ni binding |
| `TP-PRF-011` | bind sin perfil existente falla con `ProfileNotFound` |
| `TP-PRF-012` | list/show/current/remove operan sobre `projects.json` |
| `TP-PRF-017` | bind de perfil existente no autorizado falla antes de escribir |
| `TP-PRF-018` | bind de perfil QA existente autorizado persiste el binding |
