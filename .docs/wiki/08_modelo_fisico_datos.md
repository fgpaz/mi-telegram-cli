# 1. Alcance fisico

La v1 usa persistencia local por perfil. El modelo físico existe para proteger aislamiento, reutilización de sesión y operación segura; no introduce una base de datos compartida entre cuentas.

## 2. Owner y safety stance

| Area | Owner | Safety stance |
| --- | --- | --- |
| Metadata de perfil | Proyecto | Persistencia legible y recuperable por perfil. |
| Sesión MTProto | Proyecto | Dato sensible, aislado por perfil, nunca compartido. |
| Lock operativo | Proyecto | Debe evitar concurrencia incompatible. |

## 3. Invariantes fisicos visibles

- Un perfil tiene un único root físico.
- La sesión MTProto no se comparte entre perfiles.
- Los artefactos físicos deben poder borrarse al eliminar el perfil.

## 4. Navegacion

- Storage local y layout físico: [DB-LOCAL-STORAGE](./08_db/DB-LOCAL-STORAGE.md)

## 5. Sync triggers

Actualizar `08` y `08_db/*` cuando cambien:

- layout de archivos por perfil
- mecanismo de lock
- estrategia de persistencia de cursores o estado de auth

