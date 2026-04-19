# RF-MSG-001 - Leer mensajes recientes enriquecidos

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-MSG-001` |
| Titulo | Leer mensajes recientes enriquecidos |
| Modulo | `MSG` |
| Flow fuente | `FL-MSG-01` |
| Actor | Agente |
| Trigger | `messages read` |
| Resultado observable | Colección ordenada de `MensajeResumen` con `attachments[]` y `buttons[]` |

## 2. Detailed Preconditions

- Perfil existente y autorizado.
- Peer resuelto inequívocamente.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `peerQuery` | `string` | Sí | CLI arg | resuelve a un único peer |
| `limit` | `int` | No | CLI arg | `1..100`, default `20` |
| `afterMessageId` | `int64|null` | No | CLI arg (`--after-id`) | `>0` si se informa |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI resuelve el peer.
2. Consulta al adaptador Telegram mensajes recientes del diálogo.
3. Filtra por `afterMessageId` si corresponde.
4. Normaliza el resultado a `MensajeResumen[]` con metadata estable de adjuntos y botones inline.
5. Devuelve la colección ordenada de más reciente a más antiguo.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.items[]` | `MensajeResumen[]` | mensajes devueltos |
| `data.count` | `int` | cantidad final |
| `data.peer` | `PeerObjetivo` | peer usado |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `UnauthorizedProfile` | perfil no autorizado | `ok=false` |
| `PeerNotFound` | peer inexistente | `ok=false` |
| `PeerAmbiguous` | peer ambiguo | `ok=false` |
| `InvalidInput` | límite o `afterMessageId` inválidos | `ok=false` |
| `TelegramReadFailed` | error leyendo mensajes | `ok=false` |

## 7. Special Cases and Variants

- Si no hay mensajes, devuelve `items=[]` y `ok=true`.
- Cada `MensajeResumen` puede incluir `attachments[]` y `buttons[]` sin descargar adjuntos.
- `afterMessageId` limita la colección a mensajes posteriores y se informa por el flag público `--after-id`.

## 8. Data Model Impact

- Lee `PerfilLocal`, `EstadoAutorizacionTelegram`.
- Produce `MensajeResumen` y puede leer `CursorLectura`.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: lectura reciente exitosa
  Given el perfil "qa-dev" está autorizado
  And el peer "@multi_tedi_dev_bot" se resuelve de forma única
  When el agente ejecuta messages read con limit 10
  Then el CLI responde ok=true
  And devuelve hasta 10 mensajes resumidos enriquecidos

Scenario: peer ambiguo
  Given el perfil "qa-dev" está autorizado
  When el agente ejecuta messages read con un peer ambiguo
  Then el CLI responde ok=false con code PeerAmbiguous
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-MSG-001` | lectura reciente básica |
| `TP-MSG-002` | filtro por `afterMessageId` |
| `TP-MSG-003` | peer ambiguo |
| `TP-MSG-020` | mensaje con adjuntos y botones |
| `TP-MSG-021` | clasificación de adjuntos document/photo/voice |

## 11. No Ambiguities Left

- La lectura reciente no persiste mensajes como verdad de dominio.
- La ordenación visible es determinística.
- El contrato de adjuntos y botones es aditivo y estable para agentes.
