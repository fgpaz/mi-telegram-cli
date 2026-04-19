# RF-AUT-004 - Consultar identidad activa del perfil

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-AUT-004` |
| Titulo | Consultar identidad activa del perfil |
| Modulo | `AUT` |
| Flow fuente | `FL-AUT-03` |
| Actor | Operador tecnico, Agente |
| Trigger | `me` |
| Resultado observable | `accountSummary` visible y consistente con la sesion activa |

## 2. Detailed Preconditions

- El perfil existe.
- El perfil esta autorizado.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Si | CLI arg (`--profile`) | perfil existente |
| `outputMode` | `text|json` | No | CLI arg (`--json`) | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI carga el perfil solicitado.
2. Verifica que el estado local permita usar la sesion.
3. Consulta al adaptador Telegram la identidad activa.
4. Normaliza la respuesta a `accountSummary`.
5. Devuelve la identidad resumida sin exponer datos sensibles.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string` | perfil consultado |
| `data.accountSummary.id` | `int64|string` | identificador Telegram de la cuenta |
| `data.accountSummary.username` | `string|null` | username visible |
| `data.accountSummary.displayName` | `string` | nombre visible resumido |
| `data.accountSummary.phoneMasked` | `string|null` | telefono enmascarado |
| `data.accountSummary.isBot` | `bool` | tipo de identidad |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `ProfileNotFound` | perfil inexistente | `ok=false` |
| `UnauthorizedProfile` | perfil sin sesion valida | `ok=false` |
| `InvalidInput` | falta configuracion runtime requerida para Telegram | `ok=false` |
| `TelegramMeFailed` | fallo consultando identidad al adaptador | `ok=false` |
| `LocalStorageFailure` | no puede leerse el estado local del perfil | `ok=false` |

## 7. Special Cases and Variants

- El comando no devuelve telefono crudo, codigo de verificacion ni blobs de sesion.
- La identidad resumida debe mantenerse compatible con `data.accountSummary` de `RF-AUT-001`.

## 8. Data Model Impact

- Lee `PerfilLocal`.
- Lee `EstadoAutorizacionTelegram`.
- No modifica entidades canonicas.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: consulta exitosa de identidad activa
  Given el perfil "qa-dev" esta autorizado
  When el operador ejecuta me para "qa-dev"
  Then el CLI responde ok=true
  And data.accountSummary contiene la identidad resumida
  And no expone secretos ni blobs de sesion

Scenario: perfil sin sesion valida
  Given el perfil "qa-dev" no esta autorizado
  When el agente ejecuta me para "qa-dev"
  Then el CLI responde ok=false con code UnauthorizedProfile
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-AUT-010` | consulta exitosa de identidad |
| `TP-AUT-011` | perfil no autorizado |
| `TP-AUT-012` | fallo del adaptador Telegram |

## 11. No Ambiguities Left

- `me` es una consulta auth-owned sobre la sesion activa del perfil.
- El output reutiliza el mismo concepto de `accountSummary` definido por `RF-AUT-001`.
