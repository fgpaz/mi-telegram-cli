# Task T4: Format + Verify

## Shared Context
**Goal:** Normalizar formato y validar la integración completa antes del cierre de trazabilidad.
**Stack:** Go toolchain, ripgrep, Markdown docs.
**Architecture:** Esta tarea no introduce comportamiento; consolida todo lo hecho en docs, app y adapter y verifica que el contrato final sea coherente.

## Task Metadata
```yaml
id: T4
depends_on:
  - T1
  - T2
  - T3
agent_type: ps-worker
files:
  - modify: cmd/mi-telegram-cli/main.go
  - modify: internal/app/messages.go
  - modify: internal/app/execute_test.go
  - modify: internal/tg/types.go
  - modify: internal/tg/gotd_client.go
  - modify: internal/tg/gotd_client_test.go
complexity: low
done_when: "go test ./..."
```

## Reference
`cmd/mi-telegram-cli/main.go:101` — verify the public help matches the contract docs exactly.

## Prompt
Run `gofmt` on every touched Go file, then run the full test suite. Finish with a ripgrep pass that verifies the docs and skill mention `messages press-button`, `attachments[]`, `buttons[]`, `RF-MSG-005`, and `FL-MSG-05` consistently. Do not add new behavior in this task; only formatting, verification, and small drift fixes uncovered by the checks.

## Skeleton
```text
gofmt -> go test -> rg consistency pass
```

## Verify
`go test ./...` -> `ok`

## Commit
`chore(cli): format and verify attachments and inline button support`
