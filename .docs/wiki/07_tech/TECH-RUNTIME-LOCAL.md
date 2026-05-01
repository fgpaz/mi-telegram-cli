# TECH-RUNTIME-LOCAL

## 1. Topologia operativa

- `mi-telegram-cli` se ejecuta como proceso corto por comando.
- En modo daemon `auto` o `required`, cada invocación Telegram coordina con el daemon local antes de tomar lock.
- Cada invocación carga un perfil, espera su turno FIFO, toma lock, ejecuta la operación y libera lock.
- `messages wait` mantiene el proceso abierto solo hasta completar reply o timeout.
- `auth login --method qr` puede mantener el proceso abierto hasta aceptar el QR o agotar su timeout total.
- La espera se resuelve dentro del mismo proceso mediante observación acotada del historial reciente del peer; no se documenta un listener MTProto persistente.
- Cuando `auth login --method qr` emite refresh de token, el CLI intenta reescribir el mismo bloque del QR si la terminal soporta control ANSI/cursor; en caso contrario agrega el nuevo bloque en append seguro.

## 2. Directorio por perfil

Ruta base sugerida:

- Windows: `%USERPROFILE%\\.mi-telegram-cli\\profiles\\<profileId>\\`

Contenido esperado:

- metadata de perfil
- estado de autorización
- sesión MTProto derivada
- lock operativo
- tickets de cola por perfil
- lease de login interactivo
- cursor de lectura si se persiste

## 2.1 Configuracion de runtime

Variables de entorno requeridas para operaciones Telegram (`auth login`, `me`, `dialogs *`, `messages *`):

- `MI_TELEGRAM_API_ID`
- `MI_TELEGRAM_API_HASH`

Reglas:

- Los comandos `profiles *`, `auth status` y `auth logout` no requieren esas variables.
- Si falta cualquiera de las variables requeridas, el CLI debe fallar temprano con un error visible de entrada invalida y sin tocar la sesion del perfil.
- Las credenciales del operador no se persisten dentro del storage del perfil.

## 3. Politica de aislamiento

- Un proceso no debe abrir dos perfiles en la misma invocación.
- Un lock activo bloquea operaciones concurrentes incompatibles.
- En modo daemon, el lock activo no falla inmediatamente: el ticket FIFO espera hasta `--queue-timeout` y recién entonces devuelve `QueueTimeout`.
- La eliminación de perfil debe invalidar sesión y metadata asociada.

## 4. Operaciones largas

- `messages wait` requiere `timeout` explícito.
- `auth login --method qr` usa un timeout total del comando y puede regenerar QR dentro de esa misma invocación.
- El render del QR privilegia legibilidad en terminal con glifos compactos; no requiere browser ni UI gráfica adicional.
- No se documentan listeners persistentes en v1.
- La estrategia concreta puede usar polling acotado o lectura equivalente del historial reciente, siempre dentro del timeout del comando.
