# 1. Resumen de arquitectura

`mi-telegram-cli` se implementa como un binario local en Go que encapsula el acceso MTProto a Telegram sobre `gotd/td`, administra perfiles locales aislados y expone una superficie CLI estable para automatización por shell. La v1 no introduce servicios remotos propios ni un daemon residente: cada comando abre el perfil, ejecuta la operación y cierra contexto; `messages wait` y `auth login --method qr` son las únicas operaciones de espera prolongada por invocación.

## 2. Project Decision Priority

Fuente de verdad para este proyecto:

1. Seguridad
2. Aislamiento
3. Correctitud
4. Confiabilidad
5. Mantenibilidad
6. Costo
7. Time-to-market

## 3. Vista de contenedores

```mermaid
flowchart LR
    A[Skill Codex o Claude] --> B[mi-telegram-cli]
    U[Operador tecnico] --> B
    B --> C[Runtime local de perfil]
    C --> D[Storage local por perfil]
    C --> E[Adaptador MTProto gotd/td]
    E --> F[Telegram]
    F --> G[Bot o sistema bajo prueba]
```

## 4. Componentes y responsabilidades

| Componente | Tipo | Responsabilidad |
| --- | --- | --- |
| `mi-telegram-cli` | Binario local | Parsea comandos, valida entradas, aplica locks y entrega salida estructurada. |
| Runtime de perfil | Limite lógico interno | Carga contexto de un perfil, resuelve peer, ejecuta operaciones y garantiza aislamiento. |
| Storage local por perfil | Persistencia local | Guarda metadata del perfil, sesión MTProto derivada, estado operativo y locks. |
| Adaptador Telegram | Integración | Traduce operaciones del CLI a llamadas MTProto vía `gotd/td`. |
| Skill de agente | Integración local | Invoca el CLI desde shell y adapta su salida al flujo de Codex/Claude. |
| Telegram | Servicio externo | Autenticación, diálogos, recepción y entrega de mensajes. |

## 5. Stack tecnologico

| Capa | Tecnologia | Razon |
| --- | --- | --- |
| CLI | Go | Binario único, fácil distribución local y buen encastre con el patrón de `mi-lsp`. |
| Cliente Telegram | `gotd/td` | Cliente MTProto maduro para usuarios y bots, con control total desde Go. |
| Persistencia local | Archivos por perfil | Minimiza dependencias externas y favorece aislamiento por cuenta. |
| Salida automatizable | JSON opcional + texto humano | Permite uso directo por shell y por skills de agentes. |
| Integración de agentes | Skill folder-based | Evita acoplar la herramienta a un protocolo MCP particular. |

## 6. Secuencia representativa: smoke E2E

```mermaid
sequenceDiagram
    participant Agent as Skill/Agente
    participant CLI as mi-telegram-cli
    participant Store as Perfil local
    participant TG as Telegram
    participant Bot as Bot objetivo

    Agent->>CLI: auth status / login
    CLI->>Store: cargar perfil y lock
    CLI->>TG: autenticar cuenta dedicada
    Agent->>CLI: messages send "/start <pairingCode>"
    CLI->>TG: enviar mensaje
    TG->>Bot: entregar comando
    Agent->>CLI: messages send "hola"
    CLI->>TG: enviar mensaje
    TG->>Bot: entregar texto
    Agent->>CLI: messages wait --timeout
    CLI->>TG: consultar historial reciente hasta reply/timeout
    TG-->>CLI: mensaje respuesta
    CLI-->>Agent: salida estructurada
```

## 7. Decisiones arquitectonicas base

- V1 CLI-first, sin MCP propio.
- V1 sin daemon local persistente.
- Un perfil = una cuenta Telegram dedicada = un storage aislado.
- `auth login` soporta código o QR de terminal sin abrir browser ni UI gráfica adicional.
- Las operaciones son síncronas por comando; `messages wait` usa espera con timeout por invocación.
- `messages wait` observa mensajes recientes del peer dentro del proceso de esa invocación y no introduce listeners persistentes ni background workers.
- `messages read` y `messages wait` exponen un `MensajeResumen` enriquecido con metadata de adjuntos y botones inline, sin descargar archivos.
- `messages press-button` opera sobre un `messageId` exacto y un selector de botón explícito; ejecuta callbacks reales y puede informar URLs visibles sin abrir UI externa.
- `messages send-photo` sube UNA foto local validada antes de tocar Telegram (existencia, extension `{jpg,jpeg,png,webp}`, cap 10 MiB) y devuelve metadata derivada (`media{kind,mimeType,sizeBytes,sha256,caption?}`) sin exponer el `filePath` local.
- El perfil `qa-alt` es estado de usuario real protegido: el guard cross-cutting rechaza con `ProfileProtected` cualquier subcomando modificador (`auth login`, `auth logout`, `dialogs mark-read`, `messages send`, `messages send-photo`, `messages press-button`) y permite los read-only para inspección humana.
- La sesión MTProto es derivada física y no redefine el modelo semántico.

## 8. Insumos para FL

### Inventario inicial de flujos

| Flow ID | Objetivo | Actores | Modulos |
| --- | --- | --- | --- |
| `FL-PRF-01` | Gestionar perfiles locales | Operador tecnico, Agente | CLI, Storage local |
| `FL-AUT-01` | Autenticar cuenta y persistir sesion | Operador tecnico, Telegram | CLI, Adaptador Telegram, Storage local |
| `FL-AUT-02` | Consultar o cerrar sesion | Operador tecnico, Agente | CLI, Storage local |
| `FL-AUT-03` | Consultar identidad activa del perfil | Operador tecnico, Agente | CLI, Adaptador Telegram |
| `FL-DLG-01` | Listar dialogos y resolver peer | Agente, Telegram | CLI, Adaptador Telegram |
| `FL-MSG-01` | Leer mensajes recientes enriquecidos | Agente, Telegram | CLI, Adaptador Telegram |
| `FL-MSG-02` | Enviar mensaje de texto | Agente, Telegram, Bot objetivo | CLI, Adaptador Telegram |
| `FL-MSG-03` | Esperar reply enriquecido para smoke E2E | Agente, Telegram, Bot objetivo | CLI, Adaptador Telegram |
| `FL-MSG-04` | Marcar dialogo como leido | Agente, Telegram | CLI, Adaptador Telegram |
| `FL-MSG-05` | Presionar boton inline de un mensaje | Agente, Telegram, Bot objetivo | CLI, Adaptador Telegram |
| `FL-MSG-06` | Enviar foto a peer | Agente, Telegram, Bot objetivo | CLI, Adaptador Telegram |
| `FL-SKL-01` | Ejecutar smoke desde skill | Agente, Bot objetivo | Skill, CLI |

### Estados, eventos y ownership

- Estados clave: `PerfilLocal`, `EstadoAutorizacionTelegram`, `LockPerfil`, `CursorLectura`.
- Eventos relevantes: `ProfileCreated`, `LoginCompleted`, `MessageSent`, `ReplyObserved`, `DialogMarkedRead`.
- Ownership:
  - CLI: validación, locking, envelope de salida.
  - Storage local: persistencia derivada y aislamiento.
  - Adaptador Telegram: ejecución MTProto.
  - Skill: orquestación del smoke.

### Bottlenecks y mitigaciones

- Mezcla accidental de sesiones entre cuentas: mitigado con storage aislado y lock por perfil.
- Peer ambiguo o mal resuelto: mitigado con una etapa explícita de resolución antes de leer/enviar.
- Timeout esperando respuesta del bot: mitigado con `messages wait --timeout` y error tipado.
- Selección ambigua del botón: mitigado con `button-index` prioritario y error tipado por texto duplicado.
- Re-login innecesario o sesión inválida: mitigado con `auth status` y reutilización controlada.

### Open questions

`0`
