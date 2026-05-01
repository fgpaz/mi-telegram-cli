# TP-SKL

## Objetivo

Validar el smoke E2E shell-driven desde una skill sobre el CLI.

| TP ID | RF | Escenario | Esperado |
| --- | --- | --- | --- |
| `TP-SKL-001` | `RF-SKL-001` | Smoke exitoso | `Passed` con evidencia de send + wait |
| `TP-SKL-002` | `RF-SKL-001` | Bot no responde | `Failed` por `WaitTimeout` |
| `TP-SKL-003` | `RF-SKL-001` | Perfil no autorizado | `Failed` temprano |
| `TP-SKL-004` | `RF-SKL-001` | Repo consumidor sin `tmp/smoke-*` | la skill usa comandos directos del CLI |
| `TP-SKL-005` | `RF-SKL-001` | `mi-telegram-cli` fuera de `PATH` pero resoluble | la skill continúa con ruta absoluta o bootstrap |
| `TP-SKL-006` | `RF-SKL-001` | Windows usa `pwsh` y peers `@...` quoted | el peer llega intacto al CLI y el recipe sigue siendo ejecutable |
| `TP-SKL-007` | `RF-SKL-001` | Otra operación ocupa la cola del perfil | `Failed` por `QueueTimeout` si vence el presupuesto |
| `TP-SKL-008` | `RF-SKL-001` | Login interactivo requiere terminal visible | la skill entrega un comando local al operador y no asume una ventana visible lanzada por el agente |
| `TP-SKL-009` | `RF-SKL-001` | Smoke cross-account entre dos perfiles dedicados | intercambio correlacionado exitoso y evidencia para ambas cuentas |
| `TP-SKL-010` | `RF-SKL-001` | El bot devuelve botones inline | la skill inspecciona `buttons[]` y elige un selector estable |
| `TP-SKL-011` | `RF-SKL-001` | El smoke necesita presionar un botón inline | la skill usa `messages press-button` y añade esa evidencia al veredicto |
| `TP-SKL-012` | `RF-SKL-001` | Git Bash en Windows con `/start <pairingCode>` | el recipe usa `MSYS_NO_PATHCONV=1` o helper equivalente y el slash-leading payload llega literal al CLI |
