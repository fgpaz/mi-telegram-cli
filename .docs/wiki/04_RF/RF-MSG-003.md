# RF-MSG-003 - Esperar reply enriquecido con timeout

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-MSG-003` |
| Titulo | Esperar reply enriquecido con timeout |
| Modulo | `MSG` |
| Flow fuente | `FL-MSG-03` |
| Actor | Agente |
| Trigger | `messages wait` |
| Resultado observable | Reply observado con `attachments[]`/`buttons[]` o timeout tipado |

## 2. Detailed Preconditions

- Perfil existente y autorizado.
- Peer resuelto inequívocamente.
- `timeoutSeconds` explicito.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `peerQuery` | `string` | Sí | CLI arg | peer resoluble |
| `afterMessageId` | `int64|null` | No | CLI arg (`--after-id`) | `>0` si se informa |
| `timeoutSeconds` | `int` | Si | CLI arg (`--timeout`) | `1..300` |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI valida el timeout.
2. Resuelve el peer y arranca la espera.
3. Escucha eventos entrantes del peer.
4. Si llega un mensaje nuevo compatible antes del timeout, lo normaliza como `MensajeResumen` enriquecido.
5. Devuelve el `MensajeResumen` recibido.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.peer` | `PeerObjetivo` | peer observado |
| `data.message` | `MensajeResumen` | reply recibido |
| `data.observedAtUtc` | `string(datetime)` | momento de observación |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `UnauthorizedProfile` | perfil no autorizado | `ok=false` |
| `PeerNotFound` | peer no resuelto | `ok=false` |
| `PeerAmbiguous` | peer ambiguo | `ok=false` |
| `InvalidInput` | timeout o `afterMessageId` inválidos | `ok=false` |
| `WaitTimeout` | no llega reply antes del timeout | `ok=false` |
| `TelegramWaitFailed` | falla de escucha/subscripción por comando | `ok=false` |

## 7. Special Cases and Variants

- Si `afterMessageId` esta presente, solo cuentan mensajes posteriores y se informa por el flag publico `--after-id`.
- El `data.message` puede incluir `attachments[]` y `buttons[]`.
- Un mensaje previo no satisface la espera.

## 8. Data Model Impact

- Lee `CursorLectura` o `afterMessageId`.
- Produce `MensajeResumen` entrante como proyección observable.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: reply observado antes del timeout
  Given el perfil "qa-dev" está autorizado
  And ya se envió un mensaje al peer objetivo
  When el agente ejecuta messages wait con timeout 30
  Then el CLI responde ok=true
  And devuelve el mensaje entrante observado enriquecido

Scenario: timeout controlado
  Given el perfil "qa-dev" está autorizado
  And no llega ningún reply nuevo
  When el agente ejecuta messages wait con timeout 5
  Then el CLI responde ok=false con code WaitTimeout
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-MSG-007` | reply observado |
| `TP-MSG-008` | timeout |
| `TP-MSG-009` | filtro por `afterMessageId` |
| `TP-MSG-022` | reply con adjuntos y botones |

## 11. No Ambiguities Left

- El timeout es obligatorio y visible.
- El RF no define listeners persistentes fuera del proceso de la invocación.
