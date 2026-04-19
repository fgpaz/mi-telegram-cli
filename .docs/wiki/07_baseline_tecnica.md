# 1. Alcance tecnico base

`mi-telegram-cli` opera como runtime local CLI-first sobre `gotd/td`, con una ejecución por comando y aislamiento estricto por perfil. La v1 no usa un daemon persistente ni un servidor MCP propio.

## 2. Componentes tecnicos canonicos

| Componente | Owner | Decision critica |
| --- | --- | --- |
| Binario CLI | Proyecto | Entrada única para usuarios y agentes. |
| Runtime de perfil | Proyecto | Aislamiento fuerte por cuenta. |
| Storage local | Proyecto | Persistencia local por perfil y locks. |
| Adaptador Telegram | Proyecto | Integración MTProto usando `gotd/td`. |
| Skill shell-driven | Proyecto | Integración con agentes sin protocolo adicional. |

## 3. Invariantes tecnicos visibles

- Un comando no reutiliza memoria compartida entre perfiles fuera del storage explícito.
- Toda operación Telegram requiere cargar el perfil y validar su estado local.
- Toda operacion que hable con Telegram requiere `MI_TELEGRAM_API_ID` y `MI_TELEGRAM_API_HASH` presentes en el entorno del proceso.
- `auth login` soporta dos modos visibles: `code` y `qr`; el modo `qr` es interactivo de terminal y no usa `--json`.
- `messages wait` es bounded por timeout y por una observación local acotada al proceso invocado; no existe espera infinita ni listener persistente en el contrato visible.
- `messages read` y `messages wait` normalizan media y reply markup a un `MensajeResumen` enriquecido con metadata estable, sin descargar adjuntos.
- `messages press-button` resuelve un mensaje exacto por `messageId` y acciona solo botones callback reales o informa URLs visibles; no abre WebView ni navegador.
- La salida automatizable usa envelope estable y no depende del formato humano.

## 4. Navegacion

- Runtime local y ciclo de ejecución: [TECH-RUNTIME-LOCAL](./07_tech/TECH-RUNTIME-LOCAL.md)
- Integración con skills: [TECH-SKILL-INTEGRATION](./07_tech/TECH-SKILL-INTEGRATION.md)

## 5. Sync triggers

Actualizar `07` y `07_tech/*` cuando cambien:

- topología de ejecución local
- uso de daemon
- política de timeout y operación larga
- mecanismo de integración con agentes
