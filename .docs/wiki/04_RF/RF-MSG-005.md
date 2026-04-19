# RF-MSG-005 - Presionar boton inline de un mensaje

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-MSG-005` |
| Titulo | Presionar boton inline de un mensaje |
| Modulo | `MSG` |
| Flow fuente | `FL-MSG-05` |
| Actor | Agente |
| Trigger | `messages press-button` |
| Resultado observable | Callback ejecutado o URL visible informada |

## 2. Detailed Preconditions

- Perfil existente y autorizado.
- Peer resuelto inequívocamente.
- `messageId` visible del mensaje objetivo.
- Se informa `buttonIndex` o `buttonText`.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `peerQuery` | `string` | Sí | CLI arg | peer resoluble |
| `messageId` | `int64` | Sí | CLI arg (`--message-id`) | `>0` |
| `buttonIndex` | `int|null` | No | CLI arg (`--button-index`) | `>=0` |
| `buttonText` | `string|null` | No | CLI arg (`--button-text`) | texto no vacío |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI valida `messageId` y el selector del botón.
2. Resuelve el peer y recupera el mensaje exacto.
3. Normaliza los botones inline visibles del mensaje.
4. Si `buttonIndex` y `buttonText` están presentes, prioriza `buttonIndex`.
5. Si el botón es callback, ejecuta `messages.getBotCallbackAnswer`.
6. Si el botón es URL, informa la URL visible sin abrir UI externa.
7. Devuelve el resultado estructurado con el botón seleccionado.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.peer` | `PeerObjetivo` | peer usado |
| `data.action` | `callback|url` | acción observable |
| `data.button` | `InlineButtonSummary` | botón resuelto |
| `data.callbackAnswer` | `CallbackAnswerSummary|null` | respuesta del bot al callback |
| `data.url` | `string|null` | URL visible del botón |
| `data.observedAtUtc` | `string(datetime)` | marca temporal de la operación |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `UnauthorizedProfile` | perfil no autorizado | `ok=false` |
| `PeerNotFound` | peer no resuelto | `ok=false` |
| `PeerAmbiguous` | peer ambiguo | `ok=false` |
| `InvalidInput` | `messageId` o selector inválido | `ok=false` |
| `MessageNotFound` | mensaje inexistente | `ok=false` |
| `ButtonNotFound` | botón no encontrado | `ok=false` |
| `ButtonAmbiguous` | texto coincide con varios botones | `ok=false` |
| `ButtonUnsupported` | tipo de botón no compatible con el CLI | `ok=false` |
| `ButtonPasswordRequired` | callback requiere password/SRP | `ok=false` |
| `TelegramCallbackFailed` | falla accionando el callback | `ok=false` |

## 7. Special Cases and Variants

- `buttonIndex` es el selector canónico para agentes.
- `buttonText` existe como selector alternativo humano.
- Los botones URL retornan éxito con `action=url`.
- Botones WebView, request-phone, request-geo, game y similares no se accionan en esta iteración.

## 8. Data Model Impact

- Consume `MensajeResumen` y `PeerObjetivo`.
- No crea nuevas entidades canónicas.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: callback exitoso por índice
  Given el perfil "qa-dev" está autorizado
  And existe un mensaje con un botón callback inline
  When el agente ejecuta messages press-button con message-id y button-index
  Then el CLI responde ok=true
  And data.action es callback

Scenario: URL visible informada
  Given el perfil "qa-dev" está autorizado
  And existe un mensaje con un botón URL inline
  When el agente ejecuta messages press-button con button-text
  Then el CLI responde ok=true
  And data.action es url
  And data.url contiene la URL visible

Scenario: selector ambiguo por texto
  Given el perfil "qa-dev" está autorizado
  And el mensaje contiene dos botones con el mismo texto
  When el agente ejecuta messages press-button con button-text
  Then el CLI responde ok=false con code ButtonAmbiguous
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-MSG-023` | callback exitoso por índice |
| `TP-MSG-024` | URL visible informada |
| `TP-MSG-025` | selector ambiguo |
| `TP-MSG-026` | botón no soportado |
| `TP-MSG-027` | mensaje no encontrado |
| `TP-MSG-028` | callback requiere password |
| `TP-MSG-029` | botón inexistente |
| `TP-MSG-030` | falla callback genérica |

## 11. No Ambiguities Left

- El comando opera sobre un `messageId` exacto.
- El selector por índice gana sobre texto.
- La v1.1 no abre navegador ni WebView.
