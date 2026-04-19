# RF-AUT-001 - Iniciar login por codigo o QR terminal

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-AUT-001` |
| Titulo | Iniciar login por codigo o QR terminal |
| Modulo | `AUT` |
| Flow fuente | `FL-AUT-01` |
| Actor | Operador tecnico |
| Trigger | `auth login` |
| Resultado observable | Perfil autorizado y reutilizable |

## 2. Detailed Preconditions

- El perfil existe.
- No existe lock incompatible.
- La cuenta objetivo es una cuenta dedicada de QA.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | perfil existente |
| `loginMethod` | `code|qr` | No | CLI arg/interaccion | si se omite en terminal interactiva, el CLI solicita `QR` o `Phone + code`; fuera de TTY o con flags de login por codigo presentes, default `code` |
| `phoneNumber` | `string|null` | Cond | CLI arg/config | requerido para `loginMethod=code` |
| `verificationCode` | `string|null` | Cond | CLI arg/interaccion posterior al `SendCode` | requerido para completar `loginMethod=code` |
| `twoFactorPassword` | `string|null` | No | CLI arg/interaccion posterior al challenge 2FA | solo si la cuenta lo exige en `loginMethod=code` |
| `timeoutSeconds` | `int|null` | No | CLI arg | aplica a `loginMethod=qr`, default `120`, `>0` |
| `outputMode` | `text|json` | No | CLI arg | `json` no aplica a `loginMethod=qr` |

## 4. Process Steps (Happy Path)

1. El CLI toma lock del perfil.
2. Carga metadata del perfil.
3. Resuelve `loginMethod`: usa el valor explicito si existe; si falta y la terminal es interactiva, solicita `QR` o `Phone + code`; si falta fuera de TTY o ya hay flags de login por codigo, usa `code`.
4. Si `loginMethod=code`, valida `phoneNumber`, ejecuta `SendCode` contra Telegram y deja emitido el challenge de login dentro de la misma invocación.
5. Si `loginMethod=code`, consume `verificationCode` desde flag o prompt solo después del `SendCode`; si Telegram exige 2FA y no llegó `twoFactorPassword`, lo solicita en la misma invocación.
6. Si `loginMethod=qr`, muestra un QR compacto de terminal y deep link de respaldo, con refresh automático hasta timeout total o aceptación.
7. Completa autenticación y recibe sesión válida.
8. Persiste `EstadoAutorizacionTelegram=Authorized` y la sesión derivada.
9. Libera lock y devuelve éxito.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string` | perfil autenticado |
| `data.authorizationStatus` | `Authorized` | estado final |
| `data.authorizedAtUtc` | `string(datetime)` | marca temporal |
| `data.accountSummary` | `object` | identidad resumida sin secretos |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `ProfileNotFound` | perfil inexistente | `ok=false` |
| `ProfileLocked` | lock activo | `ok=false` |
| `InvalidInput` | teléfono/código inválidos | `ok=false` |
| `InvalidVerificationCode` | código rechazado | `ok=false`, sin autorización parcial |
| `AuthQrTimeout` | QR no aceptado dentro del timeout total | `ok=false`, sin autorización parcial |
| `TelegramAuthFailed` | fallo general de auth/2FA | `ok=false` |
| `LocalStorageFailure` | fallo persistiendo sesión/estado | `ok=false`, no marcar autorizado |

## 7. Special Cases and Variants

- Si la cuenta exige 2FA, el flujo puede solicitar `twoFactorPassword` en la misma invocación después de que Telegram confirme el challenge.
- Si `loginMethod=qr`, el CLI opera como flujo interactivo de terminal y no expone `--json`.
- Si `loginMethod=qr`, el QR se regenera dentro de la misma invocación hasta agotar `timeoutSeconds`.
- Si `loginMethod=code` y `--code` no viene informado, el CLI lo pide solo después de que `SendCode` haya sido aceptado por Telegram.
- Si la terminal soporta control ANSI/cursor, el refresh del QR reescribe el mismo bloque visible; si no, el CLI agrega un nuevo bloque sin perder el flujo.
- Si `loginMethod` se omite y ya existen `--json`, `--phone`, `--code` o `--password`, el CLI infiere `code` y no muestra el prompt de seleccion.
- Si ya existe una sesión válida y el operador decide reloguear, la sesión previa se reemplaza al cierre exitoso.

## 8. Data Model Impact

- Lee `PerfilLocal`.
- Crea o actualiza `EstadoAutorizacionTelegram`.
- Persiste sesión MTProto derivada.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: login exitoso
  Given existe el perfil "qa-dev"
  And el operador dispone de un código válido
  When ejecuta auth login para "qa-dev"
  Then el CLI responde ok=true
  And el perfil queda en estado Authorized

Scenario: código inválido
  Given existe el perfil "qa-dev"
  When el operador completa auth login con un código inválido
  Then el CLI responde ok=false con code InvalidVerificationCode
  And el perfil no queda autorizado

Scenario: login QR exitoso
  Given existe el perfil "qa-dev"
  And el operador dispone de otra sesión Telegram capaz de escanear el QR
  When ejecuta auth login para "qa-dev" con loginMethod qr
  Then el CLI responde auth ok
  And el perfil queda en estado Authorized

Scenario: timeout en login QR
  Given existe el perfil "qa-dev"
  And el operador no acepta el QR a tiempo
  When ejecuta auth login para "qa-dev" con loginMethod qr
  Then el CLI responde ok=false con code AuthQrTimeout
  And el perfil no queda autorizado

Scenario: seleccion interactiva del metodo
  Given existe el perfil "qa-dev"
  And el operador ejecuta auth login sin loginMethod en una terminal interactiva
  When selecciona "QR" o "Phone + code"
  Then el CLI ejecuta el flujo correspondiente sin requerir relanzar el comando
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-AUT-001` | login exitoso |
| `TP-AUT-002` | código inválido |
| `TP-AUT-003` | 2FA requerida |
| `TP-AUT-013` | login QR exitoso |
| `TP-AUT-014` | refresh automático de QR |
| `TP-AUT-015` | timeout total de QR |
| `TP-AUT-016` | flags incompatibles en modo QR |
| `TP-AUT-017` | prompt interactivo selecciona QR |
| `TP-AUT-018` | prompt interactivo selecciona code |
| `TP-AUT-019` | flags de login por codigo omiten prompt |
| `TP-AUT-020` | seleccion invalida reintenta prompt |

## 11. No Ambiguities Left

- El login exitoso por código o QR es la única condición que habilita operaciones Telegram posteriores.
- La sesión serializada nunca se devuelve en output.
