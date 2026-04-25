# RF-MSG-006 - Enviar foto a peer

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-MSG-006` |
| Titulo | Enviar foto a peer |
| Modulo | `MSG` |
| Flow fuente | `FL-MSG-06` |
| Actor | Agente |
| Trigger | `messages send-photo` |
| Resultado observable | Confirmacion de envio con metadata derivada de la foto saliente |

## 2. Detailed Preconditions

- Perfil existente y autorizado.
- Perfil distinto de `qa-alt` (estado de usuario real protegido contra automatizacion).
- Peer resuelto inequivocamente.
- Archivo local existente, no vacio y dentro del cap soportado (<= 10 MiB).
- Extension permitida: `.jpg`, `.jpeg`, `.png`, `.webp` (case-insensitive).

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Si | CLI arg | perfil existente; rechazado si es perfil protegido (`qa-alt`) |
| `peerQuery` | `string` | Si | CLI arg | peer resoluble |
| `filePath` | `string` | Si | CLI arg | path local existente, no directorio, tamano `1..10485760` bytes, extension en `{jpg,jpeg,png,webp}` |
| `caption` | `string` | No | CLI arg | `0..1024` chars |
| `outputMode` | `text\|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI valida flags y guard de perfil protegido.
2. Antes de tocar Telegram, valida el archivo local: existencia, no directorio, tamano > 0 y <= 10 MiB, extension soportada; calcula SHA256 y MIME derivado de la extension.
3. Adquiere el lock del perfil y resuelve el peer.
4. Sube la foto a Telegram via uploader y envia con `messages.sendMedia` + `inputMediaUploadedPhoto`.
5. Recibe confirmacion, extrae `messageId` y `sentAtUtc` desde el `Updates` retornado.
6. Devuelve `data.peer`, `data.messageId`, `data.sentAtUtc` y `data.media{kind, mimeType, sizeBytes, sha256[, caption]}`. **El `filePath` original nunca aparece en el output**.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.peer` | `PeerObjetivo` | peer usado |
| `data.messageId` | `int64\|string` | identificador del mensaje saliente |
| `data.sentAtUtc` | `string(datetime)` | timestamp del envio |
| `data.media.kind` | `string` | siempre `photo` en v1 |
| `data.media.mimeType` | `string` | derivado de la extension (`image/jpeg`, `image/png`, `image/webp`) |
| `data.media.sizeBytes` | `int64` | tamano del archivo local enviado |
| `data.media.sha256` | `string` | digest hex (64 chars) sobre el contenido enviado |
| `data.media.caption` | `string` | omitido si llego vacio |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `UnauthorizedProfile` | perfil no autorizado | `ok=false` |
| `PeerNotFound` | peer no resuelto | `ok=false` |
| `PeerAmbiguous` | peer ambiguo | `ok=false` |
| `InvalidInput` | flags ausentes, caption > 1024 chars, archivo vacio, archivo > 10 MiB, path es directorio | `ok=false` |
| `FileNotFound` | archivo local inexistente | `ok=false` |
| `UnsupportedMediaType` | extension fuera de `{jpg,jpeg,png,webp}` | `ok=false` |
| `TelegramSendPhotoFailed` | Telegram rechaza el upload o el `sendMedia` | `ok=false` |
| `ProfileProtected` | `--profile qa-alt` (cross-cutting) | `ok=false` |
| `ProfileLocked` | otra operacion ya posee el lock del perfil | `ok=false` |

## 7. Special Cases and Variants

- v1 envia exactamente UNA foto por invocacion. Albums (`grouped_id`) no soportados.
- v1 NO descarga la foto enviada para verificacion; el `messageId` es la prueba observable.
- El `sha256` se calcula sobre el archivo local antes del upload; sirve como huella estable de "que foto se mando".
- El `filePath` jamas se expone en `data` ni en mensajes de error: la regla protege paths sensibles que pueden contener nombres de fixtures, directorios temporales o estructura local.
- `--caption` no soporta entidades Telegram (parse mode); se envia como texto plano.
- `qa-alt` queda bloqueado por el guard cross-cutting; el operador humano debe usar otro perfil para enviar fotos automaticas.

## 8. Data Model Impact

- Produce `MensajeResumen` saliente con `attachments[]` derivado del envio (proyeccion observable, no persistida en el storage local).
- No modifica entidades canonicas persistentes.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: envio exitoso de foto con caption
  Given el perfil "qa-dev" esta autorizado
  And el peer "@multi_tedi_dev_bot" se resuelve de forma unica
  And el archivo local "vis.jpg" pesa 480 KB y existe
  When el agente ejecuta messages send-photo con --file vis.jpg --caption "qa-dev VIS"
  Then el CLI responde ok=true
  And devuelve un messageId visible
  And devuelve data.media.kind == "photo"
  And devuelve data.media.sha256 con 64 caracteres hex
  And el output JSON no contiene el path local

Scenario: archivo inexistente
  Given el perfil "qa-dev" esta autorizado
  When el agente ejecuta messages send-photo con --file ".\faltante.png"
  Then el CLI responde ok=false con code FileNotFound
  And no llama al adaptador Telegram

Scenario: extension no soportada
  Given el perfil "qa-dev" esta autorizado
  When el agente ejecuta messages send-photo con --file animado.gif
  Then el CLI responde ok=false con code UnsupportedMediaType

Scenario: archivo excede 10 MiB
  Given el perfil "qa-dev" esta autorizado
  When el agente ejecuta messages send-photo con un archivo de 11 MiB
  Then el CLI responde ok=false con code InvalidInput
  And el mensaje menciona "10MiB"

Scenario: automatizacion sobre qa-alt rechazada
  Given se ejecuta cualquier subcomando modificador con --profile qa-alt
  When el CLI evalua el guard de perfil protegido
  Then responde ok=false con code ProfileProtected
  And el mensaje dice "qa-alt is protected real-user state; use qa-dev for automation"
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-MSG-032` | envio exitoso con caption y metadata media |
| `TP-MSG-033` | caption omitido cuando llega vacio |
| `TP-MSG-034` | archivo inexistente -> FileNotFound, sin llamada al adaptador |
| `TP-MSG-035` | archivo > 10 MiB -> InvalidInput con mencion del cap |
| `TP-MSG-036` | extension fuera del set permitido -> UnsupportedMediaType |
| `TP-MSG-037` | flags faltantes (profile, peer, file) -> InvalidInput |
| `TP-MSG-038` | el output JSON nunca contiene el path local ni la temp dir |
| `TP-MSG-039` | `ProfileLocked` cuando otra operacion ya posee el lock |
| `TP-MSG-040` | guard cross-cutting `qa-alt` rechaza modificadores y permite read-only |

## 11. No Ambiguities Left

- v1 cubre solo upload de UNA foto desde disco.
- Albums, documentos, video, voice y stickers quedan fuera de scope.
- El cap de tamano (10 MiB) coincide con el limite Telegram para `inputMediaUploadedPhoto` y se valida antes de abrir el adaptador.
- El SHA256 NO sustituye al `messageId` como identificador remoto: es solo la huella local de la foto enviada.
