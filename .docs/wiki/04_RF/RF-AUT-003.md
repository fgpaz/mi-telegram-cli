# RF-AUT-003 - Cerrar sesión local

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-AUT-003` |
| Titulo | Cerrar sesión local |
| Modulo | `AUT` |
| Flow fuente | `FL-AUT-02` |
| Actor | Operador tecnico |
| Trigger | `auth logout` |
| Resultado observable | Sesión local invalidada |

## 2. Detailed Preconditions

- El perfil existe.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI toma lock del perfil.
2. Carga estado y sesión local.
3. Invalida la sesión local.
4. Actualiza `EstadoAutorizacionTelegram` a `LoggedOut`.
5. Devuelve confirmación.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string` | perfil afectado |
| `data.authorizationStatus` | `LoggedOut|Unauthorized` | estado final |
| `data.sessionRemoved` | `bool` | `true` cuando había sesión |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `ProfileNotFound` | perfil inexistente | `ok=false` |
| `ProfileLocked` | lock activo | `ok=false` |
| `LocalStorageFailure` | no puede invalidar sesión | `ok=false` |

## 7. Special Cases and Variants

- Si no hay sesión vigente, el logout es idempotente y devuelve éxito.

## 8. Data Model Impact

- Actualiza `EstadoAutorizacionTelegram`.
- Elimina o invalida la sesión MTProto derivada.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: logout con sesión activa
  Given el perfil "qa-dev" está autorizado
  When el operador ejecuta auth logout para "qa-dev"
  Then el CLI responde ok=true
  And data.authorizationStatus es LoggedOut

Scenario: logout idempotente
  Given el perfil "qa-dev" no tiene sesión vigente
  When el operador ejecuta auth logout para "qa-dev"
  Then el CLI responde ok=true
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-AUT-007` | logout con sesión |
| `TP-AUT-008` | logout idempotente |
| `TP-AUT-009` | lock concurrente |

## 11. No Ambiguities Left

- El logout relevante para el MVP es local.
- La operación no deja una sesión reutilizable después del éxito.

