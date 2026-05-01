# TP-AUD

## Casos

- `TP-AUD-001`: evento JSONL v1 se escribe en archivo diario.
- `TP-AUD-002`: `audit export --errors-only` filtra errores.
- `TP-AUD-003`: `audit summary` calcula conteos y percentiles.
- `TP-AUD-004`: evento no filtra cuerpos de mensaje, captions, códigos, passwords, API hash, session blobs ni paths de archivos enviados.
- `TP-AUD-005`: filtros `--since`, `--profile` y `--operation` son combinables.

## Evidencia actual

Cubierto por `internal/audit/audit_test.go` y `go test ./...`.
