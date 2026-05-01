# 1. Alcance fisico

La v1 usa persistencia local por perfil, un registro global de bindings por proyecto, estado local de daemon y auditoría por usuario. El modelo físico existe para proteger aislamiento, reutilización de sesión y operación segura; no introduce una base de datos compartida entre cuentas ni copia sesiones por proyecto.

## 2. Owner y safety stance

| Area | Owner | Safety stance |
| --- | --- | --- |
| Metadata de perfil | Proyecto | Persistencia legible y recuperable por perfil. |
| Registro `projects.json` | Proyecto | Selección automática de perfil por repo sin guardar secretos. |
| Sesión MTProto | Proyecto | Dato sensible, aislado por perfil, nunca compartido. |
| Lock operativo | Proyecto | Debe evitar concurrencia incompatible. |
| Cola y lease daemon | Proyecto | Ordena concurrencia por perfil sin duplicar sesión. |
| Auditoría JSONL | Proyecto | Diagnóstico redacted, local y diario. |

## 3. Invariantes fisicos visibles

- Un perfil tiene un único root físico.
- La sesión MTProto no se comparte entre perfiles.
- La sesión MTProto de un perfil sí se comparte entre proyectos solo cuando esos proyectos resuelven intencionalmente al mismo `profileId`.
- `projects.json` solo guarda bindings `projectRoot -> profileId`; no guarda sesiones, tokens, chat IDs ni contenido Telegram.
- Repos concurrentes importantes deben usar perfiles distintos, por ejemplo `multi-tedi -> qa-multi-tedi` y `salud -> qa-salud`.
- Los tickets de cola y leases se eliminan al completar, expirar o vencer timeout.
- La auditoría nunca persiste cuerpos de mensajes, captions, códigos, passwords, API hash, session blobs ni paths de archivos enviados.
- Los artefactos físicos deben poder borrarse al eliminar el perfil.

## 4. Navegacion

- Storage local y layout físico: [DB-LOCAL-STORAGE](./08_db/DB-LOCAL-STORAGE.md)
- Auditoría daemon: [DB-DAEMON-AUDIT](./08_db/DB-DAEMON-AUDIT.md)

## 5. Sync triggers

Actualizar `08` y `08_db/*` cuando cambien:

- layout de archivos por perfil
- mecanismo de lock
- estrategia de persistencia de cursores o estado de auth
