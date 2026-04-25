# TP-MSG

## Objetivo

Validar lectura enriquecida, envío, espera enriquecida, presión de botones inline y mark-read.

| TP ID | RF | Escenario | Esperado |
| --- | --- | --- | --- |
| `TP-MSG-001` | `RF-MSG-001` | Lectura reciente básica | mensajes resumidos |
| `TP-MSG-002` | `RF-MSG-001` | Lectura con `afterMessageId` | colección filtrada |
| `TP-MSG-003` | `RF-MSG-001` | Peer ambiguo | `PeerAmbiguous` |
| `TP-MSG-020` | `RF-MSG-001` | Lectura con adjuntos y botones | `MensajeResumen` incluye `attachments[]` y `buttons[]` |
| `TP-MSG-021` | `RF-MSG-001` | Clasificación de adjuntos | `photo/document/video/voice/audio/sticker/...` correctos |
| `TP-MSG-004` | `RF-MSG-002` | Envío exitoso | `messageId` visible |
| `TP-MSG-005` | `RF-MSG-002` | Texto inválido | `InvalidInput` |
| `TP-MSG-006` | `RF-MSG-002` | Falla de envío | `TelegramSendFailed` |
| `TP-MSG-031` | `RF-MSG-002` | Texto sospechoso reescrito por MSYS en modo humano | warning diagnóstico por `stderr` y envío sin alterar el contrato de salida |
| `TP-MSG-007` | `RF-MSG-003` | Reply observado | success con `MensajeResumen` |
| `TP-MSG-008` | `RF-MSG-003` | Timeout | `WaitTimeout` |
| `TP-MSG-009` | `RF-MSG-003` | Filtro por `afterMessageId` | ignora mensajes previos |
| `TP-MSG-022` | `RF-MSG-003` | Reply enriquecido | `data.message` incluye `attachments[]` y `buttons[]` |
| `TP-MSG-011` | `RF-MSG-004` | Mark-read exitoso | `markedRead=true` |
| `TP-MSG-012` | `RF-MSG-004` | Peer no encontrado | `PeerNotFound` |
| `TP-MSG-013` | `RF-MSG-004` | Idempotencia | éxito sobre diálogo ya limpio |
| `TP-MSG-023` | `RF-MSG-005` | Callback exitoso por índice | `action=callback` |
| `TP-MSG-024` | `RF-MSG-005` | Botón URL informado | `action=url`, URL visible |
| `TP-MSG-025` | `RF-MSG-005` | Selector por texto ambiguo | `ButtonAmbiguous` |
| `TP-MSG-026` | `RF-MSG-005` | Tipo de botón no soportado | `ButtonUnsupported` |
| `TP-MSG-027` | `RF-MSG-005` | Mensaje inexistente | `MessageNotFound` |
| `TP-MSG-028` | `RF-MSG-005` | Callback con password requerida | `ButtonPasswordRequired` |
| `TP-MSG-029` | `RF-MSG-005` | Botón inexistente | `ButtonNotFound` |
| `TP-MSG-030` | `RF-MSG-005` | Falla genérica del callback | `TelegramCallbackFailed` |
| `TP-MSG-032` | `RF-MSG-006` | Envío exitoso de foto con caption | `messageId` + `data.media{kind,mimeType,sizeBytes,sha256,caption}` |
| `TP-MSG-033` | `RF-MSG-006` | Caption omitido cuando viene vacío | `data.media` no contiene la clave `caption` |
| `TP-MSG-034` | `RF-MSG-006` | Archivo local inexistente | `FileNotFound` y no se llama al adaptador Telegram |
| `TP-MSG-035` | `RF-MSG-006` | Archivo excede 10 MiB | `InvalidInput` con mensaje que menciona `10MiB` |
| `TP-MSG-036` | `RF-MSG-006` | Extensión fuera del set permitido | `UnsupportedMediaType` |
| `TP-MSG-037` | `RF-MSG-006` | Flags ausentes (`profile`, `peer`, `file`) | `InvalidInput` |
| `TP-MSG-038` | `RF-MSG-006` | El output JSON nunca contiene el path local ni la temp dir | sin filtraciones |
| `TP-MSG-039` | `RF-MSG-006` | Otra operación ya posee el lock del perfil | `ProfileLocked` |
| `TP-MSG-040` | `RF-MSG-006` | Guard cross-cutting `qa-alt`: modificadores rechazados, read-only permitidos | `ProfileProtected` para login/logout/mark-read/send/send-photo/press-button; sin error para status/me/dialogs list/messages read/messages wait |
