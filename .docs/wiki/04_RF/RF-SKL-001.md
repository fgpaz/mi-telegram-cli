# RF-SKL-001 - Ejecutar smoke E2E desde skill

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-SKL-001` |
| Titulo | Ejecutar smoke E2E desde skill |
| Modulo | `SKL` |
| Flow fuente | `FL-SKL-01` |
| Actor | Agente |
| Trigger | Recipe de skill sobre shell |
| Resultado observable | Veredicto E2E con evidencia estructurada |

## 2. Detailed Preconditions

- El perfil de prueba existe y está autorizado.
- El bot objetivo es resoluble desde `dialogs list` o `peerQuery`.
- Existe un caso E2E preparado, por ejemplo un `pairingCode`.
- El binario `mi-telegram-cli` es invocable por `PATH`, ruta absoluta conocida o bootstrap desde el repo fuente.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | skill config | perfil autorizado |
| `peerQuery` | `string` | Sí | skill config | peer resoluble |
| `pairingCode` | `string|null` | No | script/test harness | requerido cuando el smoke incluye pairing |
| `helloText` | `string` | No | skill config | default `"hola"` |
| `waitTimeoutSeconds` | `int` | Sí | skill config | `1..300` |
| `cliInvocation` | `string` | Sí | skill/runtime | `PATH`, ruta absoluta conocida o bootstrap resuelto |
| `projectContextNeeded` | `bool` | No | skill/runtime | `true` solo si hace falta leer el repo consumidor para identificar bot, peer o pairing |
| `secondaryProfileId` | `string|null` | No | skill config | requerido solo para smoke cruzado entre dos cuentas |
| `secondaryPeerQuery` | `string|null` | No | skill config | peer resoluble del segundo perfil cuando aplica smoke cruzado |

## 4. Process Steps (Happy Path)

1. La skill resuelve cómo invocar `mi-telegram-cli`: `PATH`, ruta absoluta conocida o bootstrap desde el repo fuente.
2. Si la ejecución ocurre en Windows y requiere scripts o handoff local visible, la skill prefiere `pwsh` sobre `powershell`.
3. La skill consulta `auth status`.
4. Si el login interactivo debe verse en una terminal del operador, la skill delega un comando local `pwsh -File ...` o `mi-telegram-cli auth login ...` antes de continuar.
5. La skill verifica el peer objetivo y, en PowerShell, quotea valores `@...` para preservar el handle literal.
6. Solo si hace falta para identificar el target, revisa docs/config del repo consumidor sin exponer secretos.
7. Si corresponde, envía `/start <pairingCode>`.
8. Envía el mensaje funcional del smoke, por defecto `hola`.
9. Usa la cola FIFO del daemon para serializar por perfil; no trata `QueueTimeout` como error Telegram.
10. Si el bot devuelve botones inline y el smoke lo requiere, inspecciona `buttons[]` y ejecuta `messages press-button`, priorizando `button-index`.
11. Ejecuta `messages wait`.
12. Si el recipe es cross-account, ejecuta la segunda secuencia sobre un perfil dedicado independiente y correlaciona el intercambio con un token compartido.
13. Construye un veredicto final con evidencia de send + wait, y de `press-button` cuando aplique.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `data.status` | `Passed|Failed` | veredicto final |
| `data.steps[]` | `object[]` | evidencia resumida por comando |
| `data.lastReply` | `MensajeResumen|null` | última respuesta observada |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `SmokeSequenceFailed` | no se puede resolver una invocacion valida del CLI | `Failed` |
| `UnauthorizedProfile` | perfil no autorizado al iniciar el smoke | `Failed` |
| `PeerNotFound` | peer no resoluble | `Failed` |
| `QueueTimeout` | la cola del perfil no llegó a ejecutar dentro del presupuesto | `Failed` |
| `WaitTimeout` | no llega respuesta del bot | `Failed` |
| `ButtonUnsupported` | el smoke requiere un botón no accionable por el CLI | `Failed` |
| `SmokeSequenceFailed` | falla en cualquier otro paso obligatorio | `Failed` |

## 7. Special Cases and Variants

- El smoke puede ejecutarse con o sin `pairingCode`.
- El veredicto `Passed` exige al menos un envío exitoso y un reply observado.
- Si la skill corre dentro de un repo consumidor, no debe asumir que existen `tmp/smoke-*` en ese workspace.
- Si `mi-telegram-cli` no está en `PATH`, la skill debe intentar una ruta conocida o bootstrap antes de marcar el smoke como bloqueado.
- La skill puede identificar nombres de variables de entorno o config del repo consumidor, pero no debe exponer valores secretos.
- En Windows, la skill prefiere `pwsh` para helpers o handoff interactivo visible; no depende de `powershell.exe` en `PATH`.
- En PowerShell, peers `@username` o `@bot` se pasan quoted.
- En Git Bash / MSYS sobre Windows, cualquier `--text` que deba empezar con `/` requiere `MSYS_NO_PATHCONV=1` o un helper que lo exporte; de lo contrario el shell puede reescribir el payload antes de que llegue al CLI.
- La skill usa daemon local por defecto; `MI_TELEGRAM_CLI_DAEMON=off` conserva el fallback directo donde puede aparecer `ProfileLocked`.
- Si el smoke necesita accionar un botón inline, la skill debe inspeccionar `buttons[]` y preferir `button-index` antes que `button-text`.
- Si el operador debe ver QR, código o password en una terminal visible, la skill delega un comando local antes de reanudar el recipe.
- El smoke puede ejecutarse como recipe cross-account con dos perfiles dedicados; cuando aplica, la correlación se hace con un token compartido y cada perfil conserva su propia serialización.

## 8. Data Model Impact

- Lee `PerfilLocal`.
- Consume `PeerObjetivo` y `MensajeResumen`.
- No crea nuevas entidades canónicas.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: smoke exitoso
  Given existe un perfil autorizado para QA
  And el bot objetivo se resuelve correctamente
  When la skill ejecuta el recipe del smoke
  Then el resultado final es Passed
  And existe evidencia de send y de wait exitosos

Scenario: smoke falla por timeout
  Given existe un perfil autorizado para QA
  And el bot no responde dentro del tiempo configurado
  When la skill ejecuta el recipe del smoke
  Then el resultado final es Failed
  And el motivo dominante es WaitTimeout

Scenario: skill usada desde repo consumidor sin smoke helpers locales
  Given la skill corre fuera del repo fuente de mi-telegram-cli
  And el workspace actual no contiene tmp/smoke helpers
  When la skill ejecuta el recipe del smoke
  Then usa el binario del CLI directamente
  And no trata la ausencia de helpers como error por sí misma

Scenario: smoke falla por lock concurrente del perfil
  Given existe otra operación activa usando el mismo perfil de QA
  When la skill intenta iniciar el recipe del smoke
  Then el resultado final es Failed
  And el motivo dominante es ProfileLocked
  And la skill no lanza comandos adicionales en paralelo sobre ese perfil

Scenario: smoke cross-account exitoso
  Given existen dos perfiles dedicados y autorizados
  And cada cuenta resuelve el peer de la otra
  When la skill ejecuta el recipe cross-account con un token correlativo
  Then ambas cuentas observan el intercambio esperado
  And el resultado final es Passed
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-SKL-001` | smoke exitoso |
| `TP-SKL-002` | fallo por timeout |
| `TP-SKL-003` | fallo temprano por perfil no autorizado |
| `TP-SKL-004` | repo consumidor sin helpers locales |
| `TP-SKL-005` | binario fuera de PATH pero resoluble por ruta/bootstrap |
| `TP-SKL-006` | Windows usa `pwsh` y peers quoted en PowerShell |
| `TP-SKL-007` | lock concurrente sobre el mismo perfil |
| `TP-SKL-008` | login interactivo delegado a terminal visible del operador |
| `TP-SKL-009` | smoke cross-account con dos perfiles dedicados |
| `TP-SKL-010` | smoke con inspección de botones inline |
| `TP-SKL-011` | smoke con `messages press-button` exitoso |
| `TP-SKL-012` | smoke desde Git Bash en Windows con payload slash-leading preservado por `MSYS_NO_PATHCONV=1` |

## 11. No Ambiguities Left

- La lógica Telegram sigue viviendo en el CLI.
- La skill solo orquesta pasos y evalúa evidencia.
