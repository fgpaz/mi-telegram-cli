# RF-MSG-002 - Enviar mensaje de texto

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-MSG-002` |
| Titulo | Enviar mensaje de texto |
| Modulo | `MSG` |
| Flow fuente | `FL-MSG-02` |
| Actor | Agente |
| Trigger | `messages send` |
| Resultado observable | Confirmación de envío con referencia mínima al mensaje enviado |

## 2. Detailed Preconditions

- Perfil existente y autorizado.
- Peer resuelto inequívocamente.
- Texto no vacío.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `peerQuery` | `string` | Sí | CLI arg | peer resoluble |
| `text` | `string` | Sí | CLI arg | `1..4096`, trim no vacío |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI valida texto y resuelve peer.
2. Envía el mensaje mediante el adaptador Telegram.
3. Recibe confirmación del envío.
4. Devuelve metadata mínima del mensaje saliente.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.peer` | `PeerObjetivo` | peer usado |
| `data.messageId` | `int64|string` | identificador del mensaje saliente |
| `data.sentAtUtc` | `string(datetime)` | timestamp del envío |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `UnauthorizedProfile` | perfil no autorizado | `ok=false` |
| `PeerNotFound` | peer no resuelto | `ok=false` |
| `PeerAmbiguous` | peer ambiguo | `ok=false` |
| `InvalidInput` | texto vacío o demasiado largo | `ok=false` |
| `TelegramSendFailed` | Telegram rechaza o falla el envío | `ok=false` |

## 7. Special Cases and Variants

- No existe retry implícito en v1.
- El output no promete entrega al bot, solo envío aceptado por Telegram.
- En modo humano sin `--json`, si `text` ya llega con un prefijo sospechoso de reescritura MSYS/Git Bash (`C:/Program Files/Git/...`, `/mingw64/...`, etc.), el CLI puede emitir una advertencia por `stderr` sugiriendo `MSYS_NO_PATHCONV=1`.
- En modo `--json`, esa advertencia no se emite para preservar el envelope automatizable estable.

## 8. Data Model Impact

- Produce `MensajeResumen` saliente como proyección observable.
- No modifica entidades canónicas persistentes.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: envío exitoso
  Given el perfil "qa-dev" está autorizado
  And el peer "@multi_tedi_dev_bot" se resuelve de forma única
  When el agente ejecuta messages send con texto "hola"
  Then el CLI responde ok=true
  And devuelve un messageId visible

Scenario: texto inválido
  Given el perfil "qa-dev" está autorizado
  When el agente ejecuta messages send con texto vacío
  Then el CLI responde ok=false con code InvalidInput
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-MSG-004` | envío exitoso |
| `TP-MSG-005` | rechazo por texto inválido |
| `TP-MSG-006` | fallo de envío |
| `TP-MSG-031` | advertencia diagnóstica para texto sospechoso reescrito por MSYS en modo humano |

## 11. No Ambiguities Left

- El RF cubre solo texto plano.
- El `messageId` del output es el identificador observable de referencia para waits posteriores.
