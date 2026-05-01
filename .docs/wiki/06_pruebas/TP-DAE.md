# TP-DAE

## Casos

- `TP-DAE-001`: dos comandos del mismo perfil se serializan por cola.
- `TP-DAE-002`: perfiles distintos no comparten cola.
- `TP-DAE-003`: timeout de cola devuelve `QueueTimeout`.
- `TP-DAE-004`: modo `off` conserva fallback directo con `ProfileLocked`.
- `TP-DAE-005`: lease de `auth login` se adquiere, libera y expira.
- `TP-DAE-006`: `daemon start/status/stop` usa loopback y state local.

## Evidencia actual

Cubierto por `internal/profile/store_test.go` y `go test ./...`.
