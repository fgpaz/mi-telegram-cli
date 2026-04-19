# RF-PRF-003 - Eliminar perfil de forma segura

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-PRF-003` |
| Titulo | Eliminar perfil de forma segura |
| Modulo | `PRF` |
| Flow fuente | `FL-PRF-01` |
| Actor | Operador tecnico |
| Trigger | `profiles remove` |
| Resultado observable | Perfil purgado sin reutilizar datos sensibles |

## 2. Detailed Preconditions

- El perfil existe.
- No hay otra operación incompatible usando el perfil.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `force` | `bool` | No | CLI arg | default `false` |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI toma lock exclusivo del perfil.
2. Verifica si existe sesión local vigente.
3. Si la política lo permite, invalida sesión y metadata asociada.
4. Purga el árbol físico del perfil.
5. Devuelve confirmación de baja.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string` | perfil eliminado |
| `data.removed` | `bool` | `true` |
| `data.storageRootDeleted` | `bool` | confirmación de purga |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `ProfileNotFound` | perfil inexistente | `ok=false` |
| `ProfileLocked` | lock activo por otra operación | `ok=false` |
| `ProfileDeletionBlocked` | sesión activa y política sin `force` | `ok=false` |
| `LocalStorageFailure` | error purgando archivos | `ok=false`, sin falsa confirmación |

## 7. Special Cases and Variants

- Sin `force`, una sesión activa puede bloquear la baja.
- Con `force`, la sesión debe invalidarse antes de purgar.

## 8. Data Model Impact

- Elimina `PerfilLocal`.
- Elimina `EstadoAutorizacionTelegram`.
- Purga `LockPerfil` y cualquier cursor asociado.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: baja segura de perfil sin sesión activa
  Given existe el perfil "qa-dev" sin sesión autorizada
  When el operador ejecuta profiles remove "qa-dev"
  Then el CLI responde ok=true
  And el árbol físico del perfil queda purgado

Scenario: baja bloqueada por sesión activa
  Given existe el perfil "qa-dev" con sesión autorizada
  When el operador ejecuta profiles remove "qa-dev" sin force
  Then el CLI responde ok=false con code ProfileDeletionBlocked
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-PRF-007` | baja exitosa |
| `TP-PRF-008` | bloqueo por sesión activa |
| `TP-PRF-009` | purge completa con `force` |

## 11. No Ambiguities Left

- Eliminar perfil implica purgar metadata y sesión local.
- La política visible de `force` debe estar documentada por CLI.

