# CT-CLI-COMMANDS

## 1. Comandos visibles del MVP

| Comando | Input principal | Output principal |
| --- | --- | --- |
| `profiles add` | `--profile`, `--display-name` | perfil creado |
| `profiles list` | `--json` opcional | lista de perfiles |
| `profiles show` | `--profile` | detalle del perfil |
| `profiles remove` | `--profile`, `--force` opcional | confirmación de baja |
| `projects bind` | `--root`, `--profile`, `--create-profile` opcional, `--display-name` opcional | binding `projectRoot -> profileId` |
| `projects list` | `--json` opcional | lista de bindings |
| `projects show` | `--root` | binding de un root exacto |
| `projects current` | cwd actual | binding efectivo o fallback |
| `projects remove` | `--root` | binding removido |
| `auth login` | `--profile` opcional, `--method`, `--phone`, `--code`, `--password`, `--timeout` | estado autorizado |
| `auth status` | `--profile` opcional | estado actual |
| `auth logout` | `--profile` opcional | sesión invalidada |
| `me` | `--profile` opcional | identidad Telegram del perfil |
| `dialogs list` | `--profile` opcional, `--query`, `--limit` | lista de diálogos |
| `dialogs mark-read` | `--profile` opcional, `--peer` | confirmación |
| `messages read` | `--profile` opcional, `--peer`, `--limit`, `--after-id` | mensajes recientes enriquecidos |
| `messages send` | `--profile` opcional, `--peer`, `--text` | mensaje enviado |
| `messages send-photo` | `--profile` opcional, `--peer`, `--file`, `--caption` opcional | foto enviada con metadata derivada (`media{kind,mimeType,sizeBytes,sha256,caption?}`) |
| `messages wait` | `--profile` opcional, `--peer`, `--after-id`, `--timeout` | reply enriquecido observado o timeout |
| `messages press-button` | `--profile` opcional, `--peer`, `--message-id`, `--button-index` o `--button-text` | callback ejecutado o URL informada |
| `daemon start` | sin input requerido | daemon local iniciado o ya activo |
| `daemon status` | sin input requerido | estado local del daemon |
| `daemon stop` | sin input requerido | daemon detenido |
| `audit export` | filtros opcionales `--since`, `--profile`, `--operation`, `--errors-only` | eventos JSONL redacted |
| `audit summary` | filtros opcionales y `--json` | resumen por operación, perfil y proyecto |

## 1.1 Flags publicos comunes

- `--profile`: identificador estable del perfil local. En comandos Telegram es opcional; si falta, el CLI resuelve por binding de proyecto y luego fallback `qa-dev`.
- `--json`: fuerza el envelope estructurado `{ ok, profile, data, error }`.
- `--after-id`: cursor publico para `messages read` y `messages wait`; mapea al campo semantico `afterMessageId` documentado en RF.
- `--message-id`: identificador público del mensaje que contiene el botón inline.
- `--button-index`: selector canónico 0-based del botón inline.
- `--button-text`: selector alternativo por label exacto del botón inline.
- `--file`: path local absoluto o relativo del archivo a subir en `messages send-photo`. Tipos soportados: `jpg`, `jpeg`, `png`, `webp`. Tamaño máximo: 10 MiB.
- `--caption`: texto opcional adjunto a la foto en `messages send-photo`; máximo 1024 caracteres; sin parse mode.
- `--method`: selector de login visible para `auth login`; valores `code` o `qr`.
- `--timeout`: timeout total visible para `auth login --method qr` y `messages wait`.
- `--queue-timeout`: timeout de espera FIFO antes de ejecutar comandos daemon-routed; default 120s.
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
| `QueueTimeout` | La cola FIFO venció antes de ejecutar el comando |
| `DaemonUnavailable` | El daemon requerido no está disponible |
| `DaemonLeaseDenied` | Existe una lease activa incompatible para el perfil |
| `DaemonLeaseExpired` | La lease interactiva expiró antes de completarse |
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
| `TelegramSendPhotoFailed` | Fallo subiendo o enviando la foto via `messages.sendMedia` |
| `FileNotFound` | El path local de `--file` no existe |
| `UnsupportedMediaType` | La extensión del archivo no está en `{jpg,jpeg,png,webp}` |
| `ProfileProtected` | `--profile qa-alt` bloqueado para automatización por el guard cross-cutting |
| `ProjectBindingNotFound` | No existe binding para el root solicitado |
| `ProjectProfileMissing` | El binding efectivo apunta a un perfil inexistente |
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
- `tg_send_photo` -> `messages send-photo --json`
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
- En modo daemon `auto`, los comandos `auth status/logout`, `me`, `dialogs *` y `messages *` esperan en cola por perfil en vez de fallar inmediatamente con `ProfileLocked`.
- Resolución de perfil: `--profile` explícito gana; si falta, se usa el binding de prefijo más largo de `~/.mi-telegram-cli/projects.json` contra el `cwd`; si no hay binding, fallback legacy `qa-dev`.
- Bootstrap recomendado para repos concurrentes: `multi-tedi -> qa-multi-tedi` y `salud -> qa-salud`, seguido de `auth login --profile <id>` manual por cada perfil.
- `MI_TELEGRAM_CLI_DAEMON=off` conserva modo directo y puede devolver `ProfileLocked`; `MI_TELEGRAM_CLI_DAEMON=required` devuelve `DaemonUnavailable` si la coordinación no está disponible.
- `messages send-photo` valida el archivo local **antes** de tocar Telegram (existencia, no directorio, tamaño 1..10485760 bytes, extensión soportada) y nunca expone el `filePath` original en `data` ni en mensajes de error. La metadata observable es `data.media{kind,mimeType,sizeBytes,sha256,caption?}`.
- Guard cross-cutting `qa-alt`: el perfil `qa-alt` es estado de usuario real protegido. Los subcomandos modificadores (`auth login`, `auth logout`, `dialogs mark-read`, `messages send`, `messages send-photo`, `messages press-button`) responden `ProfileProtected` cuando reciben `--profile qa-alt`. Las lecturas (`auth status`, `me`, `dialogs list`, `messages read`, `messages wait`) siguen permitidas para inspección humana.
- Si `auth login --method qr` o el flujo por código requiere interacción visible del operador, el patrón compatible es delegar un comando local `pwsh -File ...` o `mi-telegram-cli auth login ...`.
