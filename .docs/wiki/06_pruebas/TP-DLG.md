# TP-DLG

## Objetivo

Validar discovery de diálogos y resolución inequívoca de peers.

| TP ID | RF | Escenario | Esperado |
| --- | --- | --- | --- |
| `TP-DLG-001` | `RF-DLG-001` | Listado básico | colección de `DialogoResumen` |
| `TP-DLG-002` | `RF-DLG-001` | Filtro por query | colección filtrada |
| `TP-DLG-003` | `RF-DLG-001` | Perfil no autorizado | `UnauthorizedProfile` |
| `TP-DLG-004` | `RF-DLG-002` | Resolución inequívoca | `PeerObjetivo` único |
| `TP-DLG-005` | `RF-DLG-002` | Peer inexistente | `PeerNotFound` |
| `TP-DLG-006` | `RF-DLG-002` | Peer ambiguo | `PeerAmbiguous` |

