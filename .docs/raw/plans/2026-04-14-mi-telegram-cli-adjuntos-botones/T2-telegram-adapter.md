# Task T2: Telegram Adapter Enrichment

## Shared Context
**Goal:** Enriquecer el adaptador Telegram para resumir adjuntos/botones y accionar callbacks inline.
**Stack:** Go, gotd/td, internal adapter layer.
**Architecture:** `internal/tg` es la frontera con Telegram; debe producir el read model público y encapsular la resolución exacta de mensajes y botones.

## Task Metadata
```yaml
id: T2
depends_on:
  - T0
agent_type: ps-worker
files:
  - modify: internal/tg/types.go:18-187
  - modify: internal/tg/gotd_client.go:351-1020
  - create: internal/tg/gotd_client_test.go
complexity: high
done_when: "go test ./internal/tg"
```

## Reference
`internal/tg/gotd_client.go:766-1020` — keep summary helpers close to `messageSummaryFromClass` and preserve additive compatibility.

## Prompt
Add new public types for `AttachmentSummary`, `InlineButtonSummary`, `PressButtonRequest`, `PressButtonResult`, and callback answer metadata. Extend `MessageSummary` additively. In `gotd_client.go`, classify media into stable kinds without downloading binaries, flatten inline keyboards into public button summaries with stable `index`, and implement `PressButton` by resolving the exact `message-id`, selecting by index or exact text, then calling Telegram only for real callbacks. URL buttons must return success with the URL and must not open anything. Unsupported kinds and password-protected callbacks must surface typed errors.

## Skeleton
```go
type PressButtonRequest struct {
    PeerQuery      string
    MessageID      int64
    ButtonIndex    int
    HasButtonIndex bool
    ButtonText     string
}
```

## Verify
`go test ./internal/tg` -> `ok`

## Commit
`feat(tg): add attachment summaries and inline button actions`
