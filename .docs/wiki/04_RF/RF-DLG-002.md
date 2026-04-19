# RF-DLG-002 - Resolver peer objetivo

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-DLG-002` |
| Titulo | Resolver peer objetivo |
| Modulo | `DLG` |
| Flow fuente | `FL-DLG-01` |
| Actor | Agente |
| Trigger | uso de `--peer` en operaciones de diálogo o mensajes |
| Resultado observable | `PeerObjetivo` inequívoco o error accionable |

## 2. Detailed Preconditions

- El perfil existe y está autorizado.
- El valor `peerQuery` fue provisto por el invocador.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `peerQuery` | `string` | Sí | CLI arg | `1..256`, trim no vacío |

## 4. Process Steps (Happy Path)

1. El CLI recibe `peerQuery`.
2. Consulta la colección de diálogos resolubles para el perfil.
3. Intenta resolver por username, chat id o dialog id.
4. Si encuentra un único match, construye `PeerObjetivo`.
5. Devuelve el peer resuelto al comando consumidor.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.peer.id` | `string|int64` | identificador utilizable |
| `data.peer.kind` | `user|bot|group|channel` | tipo del peer |
| `data.peer.displayName` | `string` | nombre resumido |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `UnauthorizedProfile` | perfil sin sesión válida | `ok=false` |
| `InvalidInput` | `peerQuery` vacío o inválido | `ok=false` |
| `PeerNotFound` | ningún match | `ok=false` |
| `PeerAmbiguous` | múltiples matches útiles | `ok=false` con contexto mínimo de desambiguación |

## 7. Special Cases and Variants

- La resolución por `dialog id` gana sobre búsqueda textual exacta.
- La resolución nunca selecciona arbitrariamente uno de varios matches.

## 8. Data Model Impact

- Produce `PeerObjetivo` como proyección no canónica.
- No modifica entidades persistidas.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: peer resuelto en forma inequívoca
  Given el perfil "qa-dev" está autorizado
  And existe un diálogo único para "@multi_tedi_dev_bot"
  When el agente usa ese valor como peerQuery
  Then el CLI resuelve un PeerObjetivo único

Scenario: peer ambiguo
  Given el perfil "qa-dev" está autorizado
  And existen múltiples diálogos que matchean "tedi"
  When el agente usa "tedi" como peerQuery
  Then el CLI responde ok=false con code PeerAmbiguous
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-DLG-004` | resolución inequívoca |
| `TP-DLG-005` | no encontrado |
| `TP-DLG-006` | ambiguo |

## 11. No Ambiguities Left

- La resolución es requisito previo para leer, enviar, esperar o marcar leído.
- La ambigüedad no se resuelve silenciosamente.

