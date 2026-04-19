# CT-CLI-COMMANDS

## 1. Comandos visibles del MVP

| Comando | Input principal | Output principal |
| --- | --- | --- |
| `profiles add` | `--profile`, `--display-name` | perfil creado |
| `profiles list` | `--json` opcional | lista de perfiles |
| `profiles show` | `--profile` | detalle del perfil |
| `profiles remove` | `--profile`, `--force` opcional | confirmación de baja |
| `auth login` | `--profile`, `--method`, `--phone`, `--code`, `--password`, `--timeout` | estado autorizado |
| `auth status` | `--profile` | estado actual |
| `auth logout` | `--profile` | sesión invalidada |
| `me` | `--profile` | identidad Telegram del perfil |
| `dialogs list` | `--profile`, `--query`, `--limit` | lista de diálogos |
| `dialogs mark-read` | `--profile`, `--peer` | confirmación |
| `messages read` | `--profile`, `--peer`, `--limit`, `--after-id` | mensajes recientes enriquecidos |
| `messages send` | `--profile`, `--peer`, `--text` | mensaje enviado |
| `messages wait` | `--profile`, `--peer`, `--after-id`, `--timeout` | reply enriquecido observado o timeout |
| `messages press-button` | `--profile`, `--peer`, `--message-id`, `--button-index` o `--button-text` | callback ejecutado o URL informada |

## 1.1 Flags publicos comunes

- `--profile`: identificador estable del perfil local.
- `--json`: fuerza el envelope estructurado `{ ok, profile, data, error }`.
- `--after-id`: cursor publico para `messages read` y `messages wait`; mapea al campo semantico `afterMessageId` documentado en RF.
- `--message-id`: identificador público del mensaje que contiene el botón inline.
- `--button-index`: selector canónico 0-based del botón inline.
- `--button-text`: selector alternativo por label exacto del botón inline.
- `--method`: selector de login visible para `auth login`; valores `code` o `qr`.
- `--timeout`: timeout total visible para `auth login --method qr` y `messages wait`.
- Si `auth login` se ejecuta sin `--method` en una terminal interactiva, el CLI solicita `QR` o `Phone + code`; fuera de TTY usa `code`.
- Si `auth login` se ejecuta sin `--method` pero ya incluye `--json`, `--phone`, `--code` o `--password`, el CLI infiere `code` y omite el prompt.
- En `auth login` por codigo, el CLI emite primero `SendCode` a Telegram y solo despues consume `--code` o el prompt interactivo correspondiente.
- Si Telegram exige 2FA y `--password` no llego por flag, el CLI puede solicitarlo dentro de la misma invocacion.
- `auth login --method qr` no soporta `--json`, `--phone`, `--code` ni `--password`.

## 1.2 Prerrequisitos de runtime para Telegram

- `auth login`, `me`, `dialogs *` y `messages *` requieren `MI_TELEGRAM_API_ID` y `MI_TELEGRAM_API_HASH` en el entorno.
- `profiles *`, `auth status` y `auth logout` siguen siendo locales.

## 2. Códigos de error visibles esperados

| Code | Uso previsto |
| --- | --- |
| `ProfileAlreadyExists` | Alta duplicada de perfil |
| `ProfileDeletionBlocked` | Baja insegura sin `force` |
| `ProfileNotFound` | Perfil inexistente |
| `ProfileLocked` | Perfil en uso por otra operación |
| `UnauthorizedProfile` | Perfil sin sesión válida |
| `InvalidInput` | Flags, valores o configuracion runtime invalida |
| `InvalidVerificationCode` | Código de login rechazado |
| `AuthQrTimeout` | QR no aceptado dentro del timeout total |
| `PeerNotFound` | Peer no resuelto |
| `PeerAmbiguous` | Resolución no inequívoca |
| `TelegramAuthFailed` | Fallo de autorización |
| `TelegramMeFailed` | Falla consultando la identidad activa |
| `TelegramListDialogsFailed` | Falla listando diálogos |
| `TelegramReadFailed` | Falla leyendo mensajes |
| `TelegramSendFailed` | Fallo enviando mensaje |
| `TelegramWaitFailed` | Falla esperando updates del peer |
| `MessageNotFound` | Mensaje objetivo inexistente |
| `ButtonNotFound` | Botón no encontrado |
| `ButtonAmbiguous` | Selector por texto ambiguo |
| `ButtonUnsupported` | Botón visible no compatible con la operación |
| `ButtonPasswordRequired` | Callback requiere password/SRP |
| `TelegramCallbackFailed` | Fallo ejecutando el callback del botón |
| `TelegramMarkReadFailed` | Falla marcando diálogo como leído |
| `LocalStorageFailure` | Falla leyendo o escribiendo estado local |
| `WaitTimeout` | No llegó reply dentro del timeout |
| `SmokeSequenceFailed` | La recipe E2E no logró completar todos los pasos obligatorios |

## 3. Compatibilidad para skills

Mapeo conceptual con el MCP previo:

- `tg_me` -> `me --json`
- `tg_dialogs` -> `dialogs list --json`
- `tg_dialog` / `tg_read` -> `messages read --json`
- `tg_send` -> `messages send --json`
- `tg_wait` -> `messages wait --json`
- `tg_press_button` -> `messages press-button --json`

Notas operativas de shell:

- En Windows, la integración compatible para scripts y handoff interactivo visible prefiere `pwsh`; `powershell.exe` puede no existir en `PATH`.
- En PowerShell, un `--peer` que empiece con `@` debe viajar quoted para preservar el handle literal.
- En Git Bash / MSYS sobre Windows, un `--text` que empiece con `/` puede ser reescrito por el shell antes de llegar al binario. Use `MSYS_NO_PATHCONV=1` o `pwsh` cuando necesite enviar un slash-leading payload literal, por ejemplo `/start <pairingCode>`.
- En modo humano de `messages send`, si el texto ya llega con un prefijo sospechoso de reescritura MSYS, el CLI puede emitir una advertencia diagnóstica por `stderr`. El contrato `--json` no cambia.
- `messages read` y `messages wait` exponen `attachments[]` y `buttons[]` en cada `MensajeResumen`.
- `messages press-button` prioriza `--button-index` si también llega `--button-text`.
- Los botones URL devuelven `action=url` y la URL visible, pero el CLI no abre navegador ni WebView.
- `ProfileLocked` puede aparecer incluso en `auth status`, `me`, `dialogs list` o `messages read` si otra invocación ya abrió el mismo perfil; la compatibilidad para skills exige una sola secuencia activa por perfil.
- Si `auth login --method qr` o el flujo por código requiere interacción visible del operador, el patrón compatible es delegar un comando local `pwsh -File ...` o `mi-telegram-cli auth login ...`.
