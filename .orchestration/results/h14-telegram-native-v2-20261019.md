# H14 — `mi-telegram-cli` native provider v2

## Resultado

**Source nativo `mi-mcp-provider/v2` ya publicado en `origin/main`; no reimplementé ni cambié source.** Repo correcto: `fgpaz/mi-telegram-cli`, módulo Go `mi-telegram-cli`; comprobé la atribución contra metadatos Go del ELF instalado y el source del repo. El checkout/wt anterior de nombre parecido era `mi-herdr`, no el repo de este ejecutable.

- SHA completo de source publicado: `17bc43dc07d450791e0d723e545a1ed6a99f8c03` (`feat: publish Telegram CLI management contract`).
- Contrato: `CT-CLI-V1@ef52b7ddd4a3be0261c811d277c09fc63431e31b`.
- Source real: `internal/management/management.go` (`Execute`, `currentVersion`, `writeManifest`, `doctor`, `writeError`) y `internal/management/provider-manifest.json` (13 operaciones con schemas e invocaciones), integrado antes del setup Telegram en `cmd/mi-telegram-cli/main.go`.
- IDs de objeto Git: `management.go` = `b2b4ed44fac5fe075e774c6e7725801cd5a4689e`; manifest = `c0daa8070fcb6e128a45e126dfc9d6df73bcefba`.
- La lectura estática del manifiesto confirma JSON parseable, formato v2, namespace `telegram`, binario `mi-telegram-cli` y 13 operaciones. El source usa management JSON tipado (`ok/data` y `ok=false/reason_code/detail`); el manifiesto es JSON directo, no el envelope de management. No se ejecutó el CLI.
- El publisher existente ya había integrado las operaciones de gestión nativas y los schemas al mismo binario: version, doctor, provider-manifest; los comandos de producto originales permanecen. No usé H10 generator: no era necesario para este source ya publicado, y copiar un artefacto generado no sería prueba de auto-registro.

## Source vs instalación vs caller

- **Source:** commit 17bc43d en `origin/main`, provider v2 nativo.
- **Instalación Tesla observada previamente:** `/home/tesla/.local/bin/mi-telegram-cli`, SHA-256 `952139ed3fee6279352b131ac55309c9b4e19ad2f4bd31273e51eeb864a78f6d` (inventario PATH H16 retenido). Lectura de build info local, sin ejecutar el binario, identifica módulo `mi-telegram-cli/cmd/mi-telegram-cli`, VCS revision `6813325f45fb692b7fd2ffa7b8c92aec16cbae8e`, `vcs.modified=false`, Go BuildID `Io7vThvSR4NFV_A-TLW3/vz--FzuB4B-q0hQ1MLFj/zD9gUbXpHJlsW0baKViw/LFGF9rVeW58fzD8j4sjd`. Es anterior al source 17bc; instalación v2 no está acreditada. No instalé, reinicié ni cambié PATH/settings.
- **Caller/uso:** no existe una observación real actual de invocation/consumidor en esta tarea; queda `explicit_incomplete`. No chats, credenciales, gateway Telegram TG17/C165, private APIs, restart o QA.
- Sin H14 global PASS. Source ≠ instalación ≠ caller.

## Navegación, límites y coordinación

- `mi-lsp` fue consultado para localizar source/contrato. Encontró el workspace de `mi-gateway`, pero en el checkout final la búsqueda exacta de `providerManifest` devolvió 0 resultados en el catálogo; se usó fallback tipado de build-info y lecturas precisas del source ya publicado. No se ejecutó un reindex.
- `mi-decidir consultar` expiró con timeout de la herramienta (fallo tipado); no lo convertí en gate. No se envió mensaje de buzón: el roster no tenía fila de task para mi pane `wNS:pR`, así que no era válido adoptar/enviar usando el task padre.
- El task Telegram preexistente tiene su propia rama/source commit publicado; no toqué su worktree ni reescribí sus archivos. Esta sesión usa worktree propio para reporte y publicación directa.
- No leí env/secretos; no ejecuté probes del CLI, pruebas adicionales, llamadas Telegram/API, QA de modelos, instalación, restart ni cambios de settings.

## Paths de producto source (ya publicados)

- `cmd/mi-telegram-cli/main.go`
- `internal/management/management.go`
- `internal/management/provider-manifest.json`

## Paths de reporte publicados por esta sesión

- `.orchestration/results/h14-telegram-native-v2-20261019.md`
- `.orchestration/results/h14-telegram-native-v2-20261019.json`

REPORTE-FINAL: .orchestration/results/h14-telegram-native-v2-20261019.md
LISTO
