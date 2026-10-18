# RF-PRJ-002 - Resolver perfil efectivo por proyecto

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-PRJ-002` |
| Titulo | Resolver perfil efectivo por proyecto |
| Modulo | `PRJ` |
| Flow fuente | `FL-PRJ-01` |
| Actor | Agente |
| Trigger | Comandos Telegram sin `--profile` |
| Resultado observable | Perfil efectivo estable para el `cwd` actual |

## 2. Decision Contract

- `--profile` explícito siempre gana.
- Si falta `--profile`, el CLI busca el binding cuyo `projectRoot` sea el prefijo más largo del `cwd`.
- El matching de paths es case-insensitive en Windows.
- `projects bind` solo registra perfiles QA ya existentes y autorizados; no crea perfiles ni inicia sesión.
- Si no hay binding, el fallback legacy es `qa-dev`.
- Si hay binding pero el perfil no existe, el CLI responde `ProjectProfileMissing` y no cae a `qa-dev`.

## 3. Affected Commands

- `auth login|status|logout`
- `me`
- `dialogs list|mark-read`
- `messages read|send|send-photo|wait|press-button`

## 4. Typed Errors

| Code | Trigger |
| --- | --- |
| `ProjectProfileMissing` | binding existente apunta a un perfil inexistente |
| `ProfileProtected` | perfil efectivo es protegido para una operación modificadora |
| `ProfileNotFound` | fallback o `--profile` explícito apunta a perfil inexistente |

## 5. Bootstrap Examples

```powershell
mi-telegram-cli projects bind --root C:\repos\mios\multi-tedi --profile qa-multi-tedi
mi-telegram-cli projects bind --root C:\repos\buho\salud --profile qa-salud
```

Los IDs de ejemplo representan perfiles QA previamente creados y autorizados. Si el perfil falta o no está autorizado, el operador del proyecto debe resolverlo; `bind` no crea perfiles ni inicia sesión.

## 6. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-PRF-013` | resolución por `cwd` y prefijo más largo |
| `TP-PRF-014` | override explícito con `--profile` |
| `TP-PRF-015` | binding roto devuelve `ProjectProfileMissing` |
| `TP-PRF-016` | ausencia de binding cae a `qa-dev` |
