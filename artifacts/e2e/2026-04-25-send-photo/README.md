# Live smoke evidence: messages send-photo

**Fecha:** 2026-04-25 (UTC 23:36)
**Tarea:** RF-MSG-006 — `messages send-photo`
**Profile:** `qa-dev` (NO `qa-alt`; guard `ProfileProtected` activo)
**Peer:** `@multi_tedi_dev_bot` (Multi TEDI Dev, id=8743780645)

## Comandos ejecutados

```powershell
$bin = ".\bin\mi-telegram-cli.exe"
$fixture = ".\artifacts\e2e\2026-04-25-send-photo\qa-dev-vis.png"

& $bin auth status --profile qa-dev --json
& $bin messages send-photo --profile qa-dev --peer "@multi_tedi_dev_bot" --file $fixture --caption "qa-dev VIS smoke 2026-04-25" --json
& $bin messages wait --profile qa-dev --peer "@multi_tedi_dev_bot" --after-id 5516 --timeout 60 --json
& $bin dialogs mark-read --profile qa-dev --peer "@multi_tedi_dev_bot" --json
```

## Resultados

| Paso | Archivo | OK | Observable |
|---|---|---|---|
| auth status | `01-auth-status.json` | true | `authorizationStatus: Authorized` |
| send-photo | `02-send-photo.json` | true | `messageId=5516`, `media.kind=photo`, `mimeType=image/png`, `sizeBytes=6312`, `sha256=25dc8d0df4f8c5d5a6795a9b148f217019ed586d76886f314ca0060cf63e8807`, caption presente |
| wait | `03-wait.json` | true | reply `messageId=5517` recibido a los ~6s con texto del bot reaccionando al smoke (sin attachments ni buttons) |
| mark-read | `04-mark-read.json` | true | `markedRead=true` |

## Verificaciones críticas

- **El JSON de `send-photo` NO contiene `filePath` original** (regex sobre `02-send-photo.json` por `qa-dev-vis`, `artifacts`, o `C:\\` no matchea fuera del campo `caption`). Solo expone metadata derivada (`media{kind,mimeType,sizeBytes,sha256,caption}`).
- `mimeType=image/png` deriva de la extensión `.png` de la fixture, no de magic bytes.
- `sizeBytes=6312` coincide con el tamaño real del fixture en disco.
- El bot envió respuesta de texto en ~6s, confirmando que el ingress de la foto llegó al pipeline de multi-tedi (la respuesta funcional depende del bot; no es parte del contrato del CLI).

## Fixture

`qa-dev-vis.png` (256x256, generado sintéticamente con `System.Drawing` — no es una foto real de comida; el smoke valida el INGRESS, no el reconocimiento VIS de multi-tedi).

## Sanitización

- IDs/usernames son públicos (`@multi_tedi_dev_bot` es bot público dev de multi-tedi).
- No se incluyen tokens, session blobs, ni paths sensibles.
- No se exporta el binario, solo los JSON outputs y el fixture sintético.
