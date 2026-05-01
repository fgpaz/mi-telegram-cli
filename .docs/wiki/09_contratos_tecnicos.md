# 1. Superficie contractual visible

La v1 expone contratos CLI con salida humana o JSON estructurado, daemon local loopback para coordinación y auditoría JSONL local. No hay HTTP API remota ni mensajes de integración remotos propios en el MVP.

## 2. Envelope canonico

Todo comando automatizable y no interactivo debe poder devolver:

```json
{
  "ok": true,
  "profile": "qa-dev",
  "data": {},
  "error": null
}
```

Cuando falla:

```json
{
  "ok": false,
  "profile": "qa-dev",
  "data": null,
  "error": {
    "code": "PeerNotFound",
    "message": "..."
  }
}
```

Excepción visible:

- `auth login --method qr` es un flujo interactivo de terminal y no expone `--json`.

## 3. Familias de comandos

| Familia | Contrato visible |
| --- | --- |
| `profiles` | Gestión segura de perfiles locales |
| `auth` | Login, estado y logout |
| `dialogs` | Listado y mark-read |
| `messages` | Read, send, wait, press-button |
| `me` | Identidad de la cuenta activa del perfil |
| `daemon` | Start/stop/status del coordinador local |
| `audit` | Export y summary de eventos JSONL redacted |

## 4. Navegacion

- Detalle de comandos y shapes: [CT-CLI-COMMANDS](./09_contratos/CT-CLI-COMMANDS.md)
- Contrato daemon local: [CT-DAEMON-LOCAL](./09_contratos/CT-DAEMON-LOCAL.md)
- Contrato de eventos de auditoría: [CT-AUDIT-EVENTS](./09_contratos/CT-AUDIT-EVENTS.md)
- Catálogo visible de errores: [CT-CLI-COMMANDS](./09_contratos/CT-CLI-COMMANDS.md)

## 5. Sync triggers

Actualizar `09` y `CT-*` cuando cambien:

- envelope JSON
- nombres de comandos o flags visibles
- códigos de error visibles al usuario o al agente
