# 1. Proposito del modelo

Este documento define el modelo de datos semántico canónico de `mi-telegram-cli`. Su foco es describir ownership, ciclo de vida, invariantes y frontera entre verdad de dominio, metadata operativa y derivaciones físicas. La estrategia de migración o bootstrap físico no vive aquí; se resume en `08` y su detalle en `08_db/`.

## 2. Clasificacion de artefactos

| Artefacto | Clasificacion | Canonico | Owner |
| --- | --- | --- | --- |
| `PerfilLocal` | Canonical entity | Sí | CLI |
| `EstadoAutorizacionTelegram` | Supporting entity | Sí | CLI |
| `LockPerfil` | Operational metadata | Sí | CLI |
| `CursorLectura` | Operational metadata | Sí | CLI |
| `PeerObjetivo` | Read model / projection | No | CLI |
| `DialogoResumen` | Read model / projection | No | Adaptador Telegram |
| `MensajeResumen` | Read model / projection | No | Adaptador Telegram |
| Sesión MTProto serializada | Physical-only concern | No | Storage local |

## 3. Entidades canonicas y supporting

### 3.1 `PerfilLocal`

Entidad raíz que representa una cuenta de trabajo local dentro del CLI.

| Aspecto | Definicion |
| --- | --- |
| Ownership | CLI |
| Tenancy boundary | Un perfil pertenece a una única cuenta Telegram dedicada. |
| Lifecycle | `Creado -> Configurado -> Autorizado -> Cerrado/Eliminado` |
| Invariantes | ID único, storage aislado, no comparte lock ni sesión con otro perfil. |
| Visibilidad | Operador técnico y agentes autorizados localmente. |

Campos semánticos mínimos:

- `profileId`
- `displayName`
- `storageRoot`
- `createdAtUtc`
- `status`

### 3.2 `EstadoAutorizacionTelegram`

Supporting entity asociada a un `PerfilLocal` para describir la aptitud del perfil para operar Telegram.

| Aspecto | Definicion |
| --- | --- |
| Ownership | CLI |
| Lifecycle | `Unauthorized -> PendingCode -> Authorized -> LoggedOut` |
| Invariantes | Solo un estado activo por perfil. |
| Persistido vs derivado | Persistido semánticamente; la sesión binaria sigue siendo derivación física. |

Campos semánticos mínimos:

- `profileId`
- `authorizationStatus`
- `authorizedAtUtc`
- `lastCheckedAtUtc`
- `logoutAtUtc`

## 4. Metadata operativa

### 4.1 `LockPerfil`

Metadata operativa que evita corrupción o mezcla de cuentas por concurrencia.

- No representa verdad de negocio.
- Debe existir como mecanismo visible para RF y TP porque afecta comportamiento observable.

### 4.2 `CursorLectura`

Metadata operativa para delimitar lecturas recientes y waits con `after-id`.

- No es verdad de negocio.
- Puede persistirse o recalcularse según implementación, pero RF debe tratarlo como soporte operativo.

## 5. Proyecciones y artefactos no canonicos

### 5.1 `PeerObjetivo`

Resultado de resolver username/chat id/dialog id a un peer utilizable.

- Es proyección.
- No debe convertirse en entidad persistente salvo necesidad operativa futura explícita.

### 5.2 `DialogoResumen`

Vista resumida de diálogos retornada por Telegram para descubrimiento y resolución.

- Consumida por `dialogs list`.
- No es fuente de verdad persistente local.

### 5.3 `MensajeResumen`

Vista resumida de mensajes usada por lectura, espera, presión de botones inline y smoke E2E.

- Consumida por `messages read`, `messages wait`, `messages press-button` y validaciones.
- Puede incluir metadata derivada de `attachments[]` y `buttons[]` sin convertir media o markup en entidades persistentes locales.
- No es entidad canónica del producto.

## 6. Notas de derivacion fisica

- La sesión MTProto serializada pertenece a la capa física y no redefine el modelo semántico.
- El storage local por perfil debe derivarse desde `PerfilLocal` y `EstadoAutorizacionTelegram`.
- Los locks y cursores pueden materializarse físicamente si mejoran seguridad u operación, pero siguen siendo metadata operativa.

## 7. Sync downstream

| Cambio en `05` | Debe reflejarse en |
| --- | --- |
| Nuevas entidades o estados | `04_RF`, `06_matriz_pruebas_RF`, `08_modelo_fisico_datos.md` |
| Nuevos envelopes o shapes visibles | `09_contratos_tecnicos.md` |
| Cambios de ownership o lifecycle | `03_FL`, `04_RF`, `06_pruebas/` |

## 8. RF-ready handoff

- Módulos impactados: `PRF`, `AUT`, `DLG`, `MSG`, `SKL`.
- Invariantes que RF debe imponer:
  - un perfil no comparte sesión con otro perfil
  - toda operación Telegram requiere perfil autorizado
  - peer ambiguo nunca se resuelve silenciosamente
  - `messages wait` siempre termina con reply o timeout tipado
- Non-goals explícitos:
  - persistir diálogos o mensajes como verdad de dominio
  - convertir la sesión MTProto en entidad semántica
  - descargar adjuntos o persistir binarios locales como verdad de dominio
