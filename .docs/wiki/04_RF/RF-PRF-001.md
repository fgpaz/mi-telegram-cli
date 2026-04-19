# RF-PRF-001 - Crear perfil local

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-PRF-001` |
| Titulo | Crear perfil local |
| Modulo | `PRF` |
| Flow fuente | `FL-PRF-01` |
| Actor | Operador tecnico |
| Trigger | `profiles add` |
| Resultado observable | Perfil persistido, aislado y listo para uso posterior |

## 2. Detailed Preconditions

- El `profileId` no existe previamente.
- El proceso tiene permisos para crear el root físico del perfil.
- No existe lock activo sobre el `profileId` solicitado.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string` | Sí | CLI arg | `1..64`, slug `[a-z0-9-_]+`, único |
| `displayName` | `string` | Sí | CLI arg | `1..120`, trim no vacío |
| `storageRootOverride` | `string|null` | No | CLI arg | Si existe, debe ser ruta local válida |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI valida `profileId` y `displayName`.
2. Determina el `storageRoot` efectivo.
3. Verifica inexistencia previa del perfil.
4. Crea la estructura local del perfil.
5. Persiste `PerfilLocal` con estado inicial `Created`.
6. Devuelve envelope de éxito con la metadata esencial.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string` | igual a `profileId` |
| `data.profileId` | `string` | ID creado |
| `data.displayName` | `string` | nombre visible persistido |
| `data.storageRoot` | `string` | root asignado |
| `data.status` | `Created|Configured` | estado inicial del perfil |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `InvalidInput` | `profileId` o `displayName` inválidos | `ok=false`, no crea nada |
| `ProfileAlreadyExists` | ya existe el perfil | `ok=false`, no altera estado existente |
| `ProfileLocked` | lock activo incompatible | `ok=false`, no crea nada |
| `LocalStorageFailure` | fallo creando estructura local | `ok=false`, cleanup del alta parcial |

## 7. Special Cases and Variants

- Si `storageRootOverride` no se informa, se usa el layout por defecto del proyecto.
- Si la creación falla después de crear directorios, la operación debe revertir el alta parcial.

## 8. Data Model Impact

- Crea `PerfilLocal`.
- No crea `EstadoAutorizacionTelegram` autorizado; ese estado queda para `RF-AUT-001`.
- Puede crear `LockPerfil` operativo durante la ejecución.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: alta exitosa de perfil
  Given no existe un perfil con id "qa-dev"
  When el operador ejecuta profiles add para "qa-dev"
  Then el CLI responde ok=true
  And el perfil queda persistido con storageRoot aislado

Scenario: alta rechazada por id duplicado
  Given ya existe un perfil con id "qa-dev"
  When el operador ejecuta profiles add para "qa-dev"
  Then el CLI responde ok=false con code ProfileAlreadyExists
  And no modifica el perfil existente
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-PRF-001` | alta exitosa |
| `TP-PRF-002` | rechazo por duplicado |
| `TP-PRF-003` | cleanup ante fallo de storage |

## 11. No Ambiguities Left

- `profileId` es el identificador estable del perfil.
- La creación no implica login automático.
- La unicidad del perfil es local al workspace de la herramienta.

