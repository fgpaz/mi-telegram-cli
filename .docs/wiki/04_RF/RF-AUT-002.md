# RF-AUT-002 - Consultar estado de autorización

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-AUT-002` |
| Titulo | Consultar estado de autorización |
| Modulo | `AUT` |
| Flow fuente | `FL-AUT-02` |
| Actor | Operador tecnico, Agente |
| Trigger | `auth status` |
| Resultado observable | Estado local actual del perfil |

## 2. Detailed Preconditions

- El perfil existe.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI carga el perfil solicitado.
2. Lee el estado local de autorización.
3. Devuelve el estado observable sin exponer secretos.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string` | perfil consultado |
| `data.authorizationStatus` | `Unauthorized|PendingCode|Authorized|LoggedOut` | estado actual |
| `data.lastCheckedAtUtc` | `string(datetime)|null` | última verificación |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `ProfileNotFound` | perfil inexistente | `ok=false` |
| `LocalStorageFailure` | no puede leerse el estado | `ok=false` |

## 7. Special Cases and Variants

- Un perfil recién creado puede devolver `Unauthorized`.
- Un perfil dado de baja no es consultable y cae en `ProfileNotFound`.

## 8. Data Model Impact

- Lee `PerfilLocal`.
- Lee `EstadoAutorizacionTelegram`.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: consulta de perfil autorizado
  Given el perfil "qa-dev" está autorizado
  When el agente ejecuta auth status para "qa-dev"
  Then el CLI responde ok=true
  And data.authorizationStatus es Authorized

Scenario: perfil inexistente
  Given no existe el perfil "qa-dev"
  When el operador ejecuta auth status para "qa-dev"
  Then el CLI responde ok=false con code ProfileNotFound
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-AUT-004` | consulta de autorizado |
| `TP-AUT-005` | consulta de no autorizado |
| `TP-AUT-006` | perfil inexistente |

## 11. No Ambiguities Left

- `auth status` es una consulta local del estado canónico del perfil.
- No fuerza revalidación remota en el MVP.

