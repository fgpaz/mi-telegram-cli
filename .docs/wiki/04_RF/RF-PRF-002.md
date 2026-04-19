# RF-PRF-002 - Listar y consultar perfil

## 1. Execution Sheet

| Campo | Valor |
| --- | --- |
| ID | `RF-PRF-002` |
| Titulo | Listar y consultar perfil |
| Modulo | `PRF` |
| Flow fuente | `FL-PRF-01` |
| Actor | Operador tecnico, Agente |
| Trigger | `profiles list`, `profiles show` |
| Resultado observable | Metadata visible del perfil o colección de perfiles |

## 2. Detailed Preconditions

- Para `profiles show`, el perfil solicitado existe.
- El storage local del producto es accesible en modo lectura.

## 3. Inputs

| Campo | Tipo | Req | Origen | Validacion |
| --- | --- | --- | --- | --- |
| `profileId` | `string|null` | No | CLI arg | requerido solo en `show` |
| `outputMode` | `text|json` | No | CLI arg | default `text` |

## 4. Process Steps (Happy Path)

1. El CLI identifica si la operación es `list` o `show`.
2. Lee la metadata de perfiles disponibles.
3. Si es `show`, selecciona el perfil pedido.
4. Devuelve la metadata estructurada sin exponer la sesión MTProto.

## 5. Outputs

| Campo | Tipo | Observable |
| --- | --- | --- |
| `ok` | `bool` | `true` |
| `profile` | `string|null` | perfil consultado o `null` en list |
| `data.items[]` | `PerfilLocal[]` | lista en `list` |
| `data.profileId` | `string` | detalle en `show` |
| `data.authorizationStatus` | `string` | estado visible del perfil |

## 6. Typed Errors

| Code | Trigger | Expected response |
| --- | --- | --- |
| `ProfileNotFound` | `show` sobre perfil inexistente | `ok=false` |
| `LocalStorageFailure` | fallo leyendo metadata | `ok=false` |

## 7. Special Cases and Variants

- `list` sobre storage vacío devuelve `items=[]` y `ok=true`.
- `show` no debe exponer secretos ni contenido serializado de sesión.

## 8. Data Model Impact

- Lee `PerfilLocal`.
- Lee `EstadoAutorizacionTelegram` visible asociado.
- No modifica entidades canónicas.

## 9. Expanded Acceptance Criteria (Gherkin)

```gherkin
Scenario: listado sin perfiles
  Given no hay perfiles creados
  When el operador ejecuta profiles list
  Then el CLI responde ok=true
  And data.items es una colección vacía

Scenario: consulta de perfil existente
  Given existe el perfil "qa-dev"
  When el operador ejecuta profiles show "qa-dev"
  Then el CLI responde ok=true
  And data.profileId es "qa-dev"
  And no expone la sesión serializada
```

## 10. Test Traceability

| TP ID | Cobertura |
| --- | --- |
| `TP-PRF-004` | listado vacío |
| `TP-PRF-005` | consulta de perfil existente |
| `TP-PRF-006` | rechazo por perfil inexistente |

## 11. No Ambiguities Left

- `list` y `show` son operaciones solo de lectura.
- La sesión MTProto nunca forma parte del output de consulta.

