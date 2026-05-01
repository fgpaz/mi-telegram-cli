# TP-PRF

## Objetivo

Validar la gestión segura de perfiles locales.

| TP ID | RF | Escenario | Esperado |
| --- | --- | --- | --- |
| `TP-PRF-001` | `RF-PRF-001` | Alta exitosa | Perfil creado con storage aislado |
| `TP-PRF-002` | `RF-PRF-001` | Duplicado | `ProfileAlreadyExists` |
| `TP-PRF-003` | `RF-PRF-001` | Falla de storage | rollback sin perfil parcial |
| `TP-PRF-004` | `RF-PRF-002` | Listado vacío | `items=[]` |
| `TP-PRF-005` | `RF-PRF-002` | Show de perfil existente | metadata visible sin secretos |
| `TP-PRF-006` | `RF-PRF-002` | Show inexistente | `ProfileNotFound` |
| `TP-PRF-007` | `RF-PRF-003` | Baja exitosa | perfil purgado |
| `TP-PRF-008` | `RF-PRF-003` | Baja bloqueada | `ProfileDeletionBlocked` |
| `TP-PRF-009` | `RF-PRF-003` | Baja con `force` | purge completa de sesión y metadata |
| `TP-PRF-010` | `RF-PRJ-001` | `projects bind --create-profile` | perfil creado `Unauthorized` y binding persistido |
| `TP-PRF-011` | `RF-PRJ-001` | `projects bind` sin perfil existente | `ProfileNotFound` |
| `TP-PRF-012` | `RF-PRJ-001` | list/show/current/remove | metadata de binding visible sin secretos |
| `TP-PRF-013` | `RF-PRJ-002` | comando Telegram sin `--profile` en repo vinculado | usa perfil del binding por `cwd` |
| `TP-PRF-014` | `RF-PRJ-002` | comando Telegram con `--profile` explícito | el flag gana sobre el binding |
| `TP-PRF-015` | `RF-PRJ-002` | binding apunta a perfil inexistente | `ProjectProfileMissing` |
| `TP-PRF-016` | `RF-PRJ-002` | repo sin binding | fallback legacy `qa-dev` |
