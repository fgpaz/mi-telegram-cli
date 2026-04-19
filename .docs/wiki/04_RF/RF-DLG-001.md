# RF-DLG-001 - Listar diálogos del perfil

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-DLG-001` |
| Titulo | Listar diálogos del perfil |
| Modulo | `DLG` |
| Flow fuente | `FL-DLG-01` |
| Actor | Agente |
| Trigger | `dialogs list` |
| Resultado observable | Colección de `DialogoResumen` consumible por humanos y skills |

## 2. Detailed Preconditions

- El perfil existe.
- El perfil está autorizado.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `query` | `string|null` | No | CLI arg | `1..256` si se informa |
| `limit` | `int|null` | No | CLI arg | `1..100` |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI valida perfil autorizado.
2. Solicita al adaptador Telegram la lista de diálogos.
3. Aplica filtro por `query` si corresponde.
4. Normaliza cada ítem a `DialogoResumen`.
5. Devuelve la colección resultante.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string` | perfil consultado |
| `data.items[]` | `DialogoResumen[]` | diálogos visibles |
| `data.count` | `int` | cantidad devuelta |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `ProfileNotFound` | perfil inexistente | `ok=false` |
| `UnauthorizedProfile` | perfil sin sesión válida | `ok=false` |
| `InvalidInput` | `limit` o `query` inválidos | `ok=false` |
| `TelegramListDialogsFailed` | fallo consultando Telegram | `ok=false` |

## 7. Special Cases and Variants

- Si no hay coincidencias, la operación devuelve `items=[]` y `ok=true`.
- La colección no debe incluir datos secretos ni serialización cruda del peer.

## 8. Data Model Impact

- Lee `PerfilLocal` y `EstadoAutorizacionTelegram`.
- Produce `DialogoResumen` como proyección no canónica.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: listado básico
  Given el perfil "qa-dev" está autorizado
  When el agente ejecuta dialogs list para "qa-dev"
  Then el CLI responde ok=true
  And data.items contiene diálogos resumidos

Scenario: perfil no autorizado
  Given el perfil "qa-dev" no está autorizado
  When el agente ejecuta dialogs list para "qa-dev"
  Then el CLI responde ok=false con code UnauthorizedProfile
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-DLG-001` | listado básico |
| `TP-DLG-002` | filtro por query |
| `TP-DLG-003` | rechazo por perfil no autorizado |

## 11. No Ambiguities Left

- El listado es una proyección efímera.
- El límite máximo visible del MVP queda acotado por RF.

