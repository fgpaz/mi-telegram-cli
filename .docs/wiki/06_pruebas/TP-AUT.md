# TP-AUT

## Objetivo

Validar login, consulta y logout de sesiones Telegram por perfil.

| TP ID | RF | Escenario | Esperado |
| --- | --- | --- | --- |
| `TP-AUT-001` | `RF-AUT-001` | Login exitoso | estado `Authorized` |
| `TP-AUT-002` | `RF-AUT-001` | Código inválido | `InvalidVerificationCode` |
| `TP-AUT-003` | `RF-AUT-001` | Cuenta con 2FA | flujo completo con password en la misma invocacion |
| `TP-AUT-004` | `RF-AUT-002` | Estado autorizado | `Authorized` visible |
| `TP-AUT-005` | `RF-AUT-002` | Estado no autorizado | `Unauthorized` visible |
| `TP-AUT-006` | `RF-AUT-002` | Perfil inexistente | `ProfileNotFound` |
| `TP-AUT-007` | `RF-AUT-003` | Logout con sesión activa | sesión invalidada |
| `TP-AUT-008` | `RF-AUT-003` | Logout idempotente | éxito sin sesión |
| `TP-AUT-009` | `RF-AUT-003` | Lock concurrente | `ProfileLocked` |
| `TP-AUT-010` | `RF-AUT-004` | Consulta de identidad activa | `accountSummary` visible sin secretos |
| `TP-AUT-011` | `RF-AUT-004` | Perfil no autorizado | `UnauthorizedProfile` |
| `TP-AUT-012` | `RF-AUT-004` | Falla del adaptador | `TelegramMeFailed` |
| `TP-AUT-013` | `RF-AUT-001` | Login QR exitoso | estado `Authorized` y sesión persistida |
| `TP-AUT-014` | `RF-AUT-001` | Refresh automático de QR | el mismo comando renueva el QR sin relanzar |
| `TP-AUT-015` | `RF-AUT-001` | Timeout total de QR | `AuthQrTimeout` |
| `TP-AUT-016` | `RF-AUT-001` | Flags incompatibles con QR | `InvalidInput` |
| `TP-AUT-017` | `RF-AUT-001` | Terminal interactiva sin `--method`, seleccion QR | se ejecuta login QR en la misma invocacion |
| `TP-AUT-018` | `RF-AUT-001` | Terminal interactiva sin `--method`, seleccion Phone + code | se ejecuta login por codigo en la misma invocacion y el prompt de code ocurre despues de `SendCode` |
| `TP-AUT-019` | `RF-AUT-001` | Terminal interactiva sin `--method` pero con flags de codigo | no aparece prompt de metodo y solo falta pedir `code` si no vino por flag |
| `TP-AUT-020` | `RF-AUT-001` | Seleccion invalida del metodo | el CLI repregunta hasta recibir una opcion valida |
