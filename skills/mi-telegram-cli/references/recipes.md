# Recipes

## multi-tedi Smoke

Canonical smoke sequence once the binary exists:

1. Ensure the dedicated QA profile is authorized.
2. Resolve the bot dialog or use a stable peer query.
3. Send `/start <pairingCode>` when the test case requires pairing.
4. Send `hola`.
5. Inspect the reply or recent messages for `buttons[]` when the flow depends on inline choices.
6. Use `messages press-button` when a button callback or URL step is part of the smoke.
7. Wait for reply with a bounded timeout.
8. Optionally mark the dialog as read.

Example shape:

```powershell
mi-telegram-cli auth status --profile qa-dev --json
mi-telegram-cli messages send --profile qa-dev --peer "@multi_tedi_dev_bot" --text "/start <pairingCode>" --json
mi-telegram-cli messages send --profile qa-dev --peer "@multi_tedi_dev_bot" --text "hola" --json
mi-telegram-cli messages read --profile qa-dev --peer "@multi_tedi_dev_bot" --limit 5 --json
mi-telegram-cli messages press-button --profile qa-dev --peer "@multi_tedi_dev_bot" --message-id 301 --button-index 0 --json
mi-telegram-cli messages wait --profile qa-dev --peer "@multi_tedi_dev_bot" --timeout 30 --json
mi-telegram-cli dialogs mark-read --profile qa-dev --peer "@multi_tedi_dev_bot" --json
```

PowerShell note:

- quote `@username` and `@bot` peer values so the shell passes them through correctly
- keep all commands for the same profile in series; do not parallelize even harmless checks such as `auth status`, `me`, or `messages read`
- prefer `buttons[].index` from `messages read` / `messages wait` as the selector for `messages press-button`

Success criteria:

- both sends return `ok=true`
- if a button was required, `messages press-button` returns `ok=true`
- `messages wait` returns `ok=true`
- the observed reply belongs to the same target peer

## Photo Smoke (multi-tedi VIS)

Use this when the bot must validate a real outgoing photo (Visual Image Service, food photo recognition, etc.).

Preconditions:

1. `qa-dev` is authorized (`auth status` returns `authorized`).
2. A local image fixture exists under one of the supported types (`jpg`, `jpeg`, `png`, `webp`) and is <= 10 MiB.

Example shape:

```powershell
$fixture = ".\fixtures\qa-dev-vis.jpg"
mi-telegram-cli auth status --profile qa-dev --json
$send = mi-telegram-cli messages send-photo --profile qa-dev --peer "@multi_tedi_dev_bot" --file $fixture --caption "qa-dev VIS photo smoke" --json | ConvertFrom-Json
$msgId = $send.data.messageId
mi-telegram-cli messages wait --profile qa-dev --peer "@multi_tedi_dev_bot" --after-id $msgId --timeout 60 --json
mi-telegram-cli dialogs mark-read --profile qa-dev --peer "@multi_tedi_dev_bot" --json
```

Success criteria:

- `messages send-photo` returns `ok=true` with `data.media.kind == "photo"` and a 64-char hex `sha256`.
- `messages wait` returns the bot reply (VIS analysis, validation, etc.) before the timeout.
- The output JSON of `messages send-photo` does NOT contain the local file path; only the derived `media{}` metadata and the `messageId`.
- Never use `--profile qa-alt` in this recipe; the guard returns `ProfileProtected`.

## Cross-Account Smoke

Use this when two dedicated QA accounts must exchange direct messages.

Preconditions:

1. both profiles are already authorized
2. each account can resolve the other by exact username or an existing dialog
3. commands remain serial per profile

Example shape:

```powershell
$token = "cross-e2e-$(Get-Date -Format 'yyyyMMddHHmmss')"
mi-telegram-cli messages send --profile qa-dev --peer "@gabrielpaz" --text "ping $token" --json
mi-telegram-cli messages read --profile qa-alt --peer "@tedi_responde" --limit 10 --json
mi-telegram-cli messages send --profile qa-alt --peer "@tedi_responde" --text "pong $token" --json
mi-telegram-cli messages read --profile qa-dev --peer "@gabrielpaz" --limit 10 --json
mi-telegram-cli dialogs mark-read --profile qa-dev --peer "@gabrielpaz" --json
mi-telegram-cli dialogs mark-read --profile qa-alt --peer "@tedi_responde" --json
```

Success criteria:

- the first account receives the second account's reply containing the same token
- the second account receives the first account's initial message containing the same token
- both mark-read calls return `ok=true`

## Repo Smoke Scripts

Use the repo scripts when the user wants a repeatable smoke quickly:

Windows / PowerShell:

```powershell
pwsh -File .\tmp\smoke-auth.ps1
pwsh -File .\tmp\smoke-bot.ps1 -Peer "@multi_tedi_dev_bot" -MarkRead
```

Linux / Bash:

```bash
bash ./tmp/smoke-auth.sh
bash ./tmp/smoke-bot.sh --peer "@multi_tedi_dev_bot" --mark-read
```

Git Bash on Windows note:

- if you call `mi-telegram-cli` directly from Git Bash and the text starts with `/`, prefix the invocation with `MSYS_NO_PATHCONV=1`
- `tmp/smoke-bot.sh` already exports `MSYS_NO_PATHCONV=1` internally so slash-leading commands such as `/start <pairingCode>` survive intact
- using `pwsh` is the simplest alternative when you do not want to deal with MSYS path translation

What they do:

1. `smoke-auth` pulls secrets with `mkey`, builds the CLI, ensures the profile exists, runs QR login if needed, then prints `auth status`, `me`, and optionally `dialogs list`.
2. `smoke-bot` pulls secrets with `mkey`, builds the CLI, checks auth, sends a unique smoke token, waits after the sent `messageId`, reads recent messages, and can mark the dialog as read.

Useful flags:

- PowerShell:
  - `smoke-auth.ps1 -ForceLogin -SkipDialogs`
  - `smoke-bot.ps1 -Peer "@multi_tedi_dev_bot" -TimeoutSec 90 -MarkRead`
- Bash:
  - `smoke-auth.sh --force-login --skip-dialogs`
  - `smoke-bot.sh --peer "@multi_tedi_dev_bot" --timeout-sec 90 --mark-read`

## Interactive Login Visibility

When the user must scan a QR code or type a phone verification code in a visible terminal:

- prefer asking the user to run a local `pwsh -File <helper.ps1>` or direct `mi-telegram-cli auth login ...` command
- do not rely on an agent-launched terminal window being visible on the user's desktop

## Peer Resolution Pattern

Prefer this order:

1. exact username or bot handle
2. exact dialog id when already known
3. broad query only when the command can surface `PeerAmbiguous`

Never continue after `PeerAmbiguous` without narrowing the target.

## Failure Handling

- `UnauthorizedProfile`: stop and re-run `auth status` or `auth login`
- `ProfileLocked`: another command is already using the same profile; this includes benign checks such as `auth status` or `me`; wait, narrow to one active shell per profile, and retry
- `auth login --method qr`: use only from a real terminal; it is interactive and does not support `--json`
- `AuthQrTimeout`: rerun `auth login --method qr` and rescan before the timeout window closes
- `PeerNotFound`: inspect `dialogs list`
- `PeerAmbiguous`: narrow the query
- `WaitTimeout`: treat as a failed smoke, not as a soft warning
- `ButtonUnsupported`: the visible button cannot be executed headlessly; stop and treat it as a blocked smoke
- `ButtonAmbiguous`: prefer `button-index` instead of `button-text`
