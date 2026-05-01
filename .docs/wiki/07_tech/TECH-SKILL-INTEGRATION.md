# TECH-SKILL-INTEGRATION

## 1. Patron de integracion

La skill invoca el binario por shell y consume su salida estructurada. La skill no implementa lógica Telegram; solo orquesta comandos y verifica resultados.
La integración debe funcionar tanto desde el repo fuente como desde repos consumidores.

## 2. Contrato operativo

- Cada comando automatizable debe aceptar `--json`.
- La skill puede encadenar `auth status`, `dialogs list`, `messages read`, `messages send`, `messages send-photo`, `messages wait` y `messages press-button`.
- `messages send-photo` recibe `--file <path>` (local) y `--caption <text>` opcional; la skill jamas debe asumir que el `filePath` queda accesible en `data.media`, solo `mimeType`, `sizeBytes`, `sha256` y `caption?`.
- La skill respeta el guard `ProfileProtected`: nunca invoca subcomandos modificadores (`auth login`, `auth logout`, `dialogs mark-read`, `messages send`, `messages send-photo`, `messages press-button`) con `--profile qa-alt`; las lecturas humanas sobre `qa-alt` siguen permitidas.
- El smoke E2E canónico usa una cuenta dedicada y un peer objetivo resoluble.
- Cuando el bot devuelve botones inline, la skill debe inspeccionar `buttons[]` y preferir `button-index` para accionar un botón específico.
- La lectura de mensajes puede exponer `attachments[]` y `buttons[]`; esa metadata es contractual y no requiere descargar adjuntos.
- La skill debe resolver el binario por `PATH`, ruta absoluta conocida o bootstrap desde el repo fuente antes de declarar bloqueo operativo.
- La ausencia de `tmp/smoke-*` en un repo consumidor no es un error por sí mismo; en ese caso la skill usa comandos directos del CLI.
- La skill solo inspecciona docs/config del repo consumidor cuando hace falta encontrar el bot, peer o pairing del caso bajo prueba.
- En Windows, la skill prefiere `pwsh` para wrappers locales, scripts y handoff interactivo visible; no asume `powershell.exe` en `PATH`.
- En PowerShell, peers `@username` o `@bot` se pasan quoted para evitar reinterpretación del shell.
- La skill usa por defecto la cola FIFO del daemon por perfil; `QueueTimeout` es el fallo accionable cuando la espera vence antes de ejecutar.
- Cuando el operador debe ver QR, código o password en una terminal visible, la skill delega un comando local `pwsh -File ...` o un comando directo del CLI antes de continuar.
- El smoke cross-account es válido como recipe operacional siempre que use dos perfiles dedicados y mantenga serialización independiente por perfil con un token compartido de correlación.

## 3. No objetivos v1

- No exponer herramientas MCP.
- No duplicar reglas de negocio de Telegram dentro de la skill.
- No almacenar secretos en la definición de la skill.
- No depender de que el workspace activo sea el repo fuente de `mi-telegram-cli`.
