# RF-MSG-004 - Marcar diálogo como leído

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-MSG-004` |
| Titulo | Marcar diálogo como leído |
| Modulo | `MSG` |
| Flow fuente | `FL-MSG-04` |
| Actor | Agente |
| Trigger | `dialogs mark-read` |
| Resultado observable | Confirmación de limpieza operativa del diálogo |

## 2. Detailed Preconditions

- Perfil existente y autorizado.
- Peer resuelto inequívocamente.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `peerQuery` | `string` | Sí | CLI arg | peer resoluble |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI resuelve el peer.
2. Ejecuta la acción de mark-read en Telegram.
3. Devuelve confirmación observable.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.peer` | `PeerObjetivo` | peer afectado |
| `data.markedRead` | `bool` | `true` |
| `data.completedAtUtc` | `string(datetime)` | marca temporal |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `UnauthorizedProfile` | perfil no autorizado | `ok=false` |
| `PeerNotFound` | peer no resuelto | `ok=false` |
| `PeerAmbiguous` | peer ambiguo | `ok=false` |
| `TelegramMarkReadFailed` | Telegram rechaza la acción | `ok=false` |

## 7. Special Cases and Variants

- Marcar leído sobre un diálogo ya limpio puede devolver éxito idempotente.

## 8. Data Model Impact

- Puede actualizar `CursorLectura` operativo si la implementación lo usa.
- No modifica entidades canónicas persistentes.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: mark-read exitoso
  Given el perfil "qa-dev" está autorizado
  And el peer objetivo se resuelve de forma única
  When el agente ejecuta dialogs mark-read
  Then el CLI responde ok=true
  And data.markedRead es true
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-MSG-011` | mark-read exitoso |
| `TP-MSG-012` | peer no encontrado |
| `TP-MSG-013` | idempotencia |

## 11. No Ambiguities Left

- La operación existe para limpieza operativa del smoke.
- No redefine archivado, mute ni otras acciones de diálogo.
