# Task T3: CLI Surface + Tests

## Shared Context
**Goal:** Exponer la nueva capacidad en el executor y el help del CLI, con cobertura sobre contratos y errores.
**Stack:** Go CLI app layer, JSON output helpers, test doubles.
**Architecture:** `internal/app` traduce flags a casos de uso y convierte errores del adaptador en errores públicos estables.

## Task Metadata
```yaml
id: T3
depends_on:
  - T0
agent_type: ps-worker
files:
  - modify: internal/app/messages.go:60-247
  - modify: internal/app/execute_test.go:736-1007
  - modify: cmd/mi-telegram-cli/main.go:101
complexity: medium
done_when: "go test ./internal/app"
```

## Reference
`internal/app/messages.go:209-247` — map typed adapter errors to public CLI error codes without leaking gotd details.

## Prompt
Add the `messages press-button` subcommand with validation for `--profile`, `--peer`, `--message-id`, `--button-index` and `--button-text`, making index win when both selectors are present. Reuse the existing JSON response pattern. Update the fake Telegram client and executor tests so `messages read` and `messages wait` assert enriched summaries, and add focused tests for callback success, URL success, selector precedence, and ambiguous-button errors.

## Skeleton
```go
case "press-button":
    fs := newFlagSet("messages press-button")
```

## Verify
`go test ./internal/app` -> `ok`

## Commit
`feat(cli): add messages press-button command`
