# 1. Alcance tecnico base

`mi-telegram-cli` opera como runtime local CLI-first sobre `gotd/td`, con daemon local de usuario para coordinación y aislamiento estricto por perfil. La v1 no usa servidor MCP propio ni endpoints remotos del daemon.

## 2. Componentes tecnicos canonicos

| Componente | Owner | Decision critica |
| --- | --- | --- |
| Binario CLI | Proyecto | Entrada única para usuarios y agentes. |
| Runtime de perfil | Proyecto | Aislamiento fuerte por cuenta. |
| Storage local | Proyecto | Persistencia local por perfil y locks. |
| Registro de proyectos | Proyecto | `projects.json` global selecciona perfil QA fijo por `cwd`. |
| Daemon local | Proyecto | Auto-start, cola FIFO por perfil, lease de login y estado loopback. |
| Auditoría JSONL | Proyecto | Eventos diarios redacted y summary operativo. |
| Adaptador Telegram | Proyecto | Integración MTProto usando `gotd/td`. |
| Skill shell-driven | Proyecto | Integración con agentes sin protocolo adicional. |

## 3. Invariantes tecnicos visibles

- Un comando no reutiliza memoria compartida entre perfiles fuera del storage explícito y la coordinación daemon local.
- El daemon escucha solo en `127.0.0.1`, usa token local en `daemon/state.json` y no expone admin UI ni endpoints remotos.
- `MI_TELEGRAM_CLI_DAEMON=off` conserva modo directo con `ProfileLocked`; `auto` asegura daemon y usa cola; `required` falla con `DaemonUnavailable` si no puede usar daemon.
- `--queue-timeout` y `MI_TELEGRAM_CLI_QUEUE_TIMEOUT_SECONDS` controlan la espera antes de ejecutar. El default es 120s.
- Toda operación Telegram requiere cargar el perfil y validar su estado local.
- Toda operación Telegram sin `--profile` resuelve perfil efectivo desde `projects.json` por prefijo más largo del `cwd`; sin binding usa fallback `qa-dev`.
- Un binding existente con perfil ausente falla con `ProjectProfileMissing`, sin fallback silencioso.
- Toda operacion que hable con Telegram requiere `MI_TELEGRAM_API_ID` y `MI_TELEGRAM_API_HASH` presentes en el entorno del proceso.
- `auth login` soporta dos modos visibles: `code` y `qr`; el modo `qr` es interactivo de terminal y no usa `--json`.
- `messages wait` es bounded por timeout y por una observación local acotada al proceso invocado; no existe espera infinita ni listener persistente en el contrato visible.
- `messages read` y `messages wait` normalizan media y reply markup a un `MensajeResumen` enriquecido con metadata estable, sin descargar adjuntos.
- `messages press-button` resuelve un mensaje exacto por `messageId` y acciona solo botones callback reales o informa URLs visibles; no abre WebView ni navegador.
- `messages send-photo` valida el archivo local antes de tocar Telegram (existencia, extension en `{jpg,jpeg,png,webp}`, tamano `1..10485760` bytes) y nunca expone el `filePath` original en `data` ni en errores; el SHA256 del payload subido es la huella estable observable.
- El perfil `qa-alt` esta protegido por un guard cross-cutting: los subcomandos modificadores (`auth login`, `auth logout`, `dialogs mark-read`, `messages send`, `messages send-photo`, `messages press-button`) responden `ProfileProtected` cuando se invocan con `--profile qa-alt`; los read-only (`auth status`, `me`, `dialogs list`, `messages read`, `messages wait`) siguen permitidos.
- La salida automatizable usa envelope estable y no depende del formato humano.

## 4. Navegacion

- Runtime local y ciclo de ejecución: [TECH-RUNTIME-LOCAL](./07_tech/TECH-RUNTIME-LOCAL.md)
- Daemon local y auditoría: [TECH-DAEMON-LOCAL](./07_tech/TECH-DAEMON-LOCAL.md)
- Integración con skills: [TECH-SKILL-INTEGRATION](./07_tech/TECH-SKILL-INTEGRATION.md)

## 5. Sync triggers

Actualizar `07` y `07_tech/*` cuando cambien:

- topología de ejecución local
- uso de daemon
- política de timeout y operación larga
- mecanismo de integración con agentes
- soporte de envío de media (uploader, tipos permitidos, cap de tamaño)
- lista de perfiles protegidos contra automatización o el alcance del guard `ProfileProtected`
- política de resolución de perfil por proyecto o fallback legacy
