# mi-telegram-cli Quickstart

## Current Status

As of April 3, 2026, this repo contains the documentation canon, the skill contract, and a working v1 Go CLI implementation.

That means:

- the skill is ready to install and version
- the operational workflow is defined and implemented
- the binary setup steps below are usable now

## Repo-Local Skill Source of Truth

This skill lives in:

`C:\repos\mios\mi-telegram-cli\skills\mi-telegram-cli`

To let Codex auto-discover it later:

```powershell
New-Item -ItemType Directory -Force $HOME\.codex\skills | Out-Null
Copy-Item -Recurse C:\repos\mios\mi-telegram-cli\skills\mi-telegram-cli $HOME\.codex\skills\
```

## Binary Setup

Recommended local setup:

1. Build or install the binary.
2. Put `mi-telegram-cli.exe` on `PATH`, or call it by absolute path.
3. If a consumer repo invokes the skill and `mi-telegram-cli` is not on `PATH`, treat that as a binary-resolution step, not as a hard blocker: try the source-repo binary path or build it from the source repo.
4. Export `MI_TELEGRAM_API_ID` and `MI_TELEGRAM_API_HASH` in the current shell before any Telegram command.
5. Use only existing, authorized dedicated Telegram QA profiles.
6. Bind a repo only with `projects bind --root <repo> --profile <existing-qa-profile>`; `bind` does not create profiles or log in.
7. If a profile is missing, not authorized, or protected, stop and ask the project owner. Agent-driven QA never runs `auth login` or creates a profile; do not use Gabriel's account.
9. On Windows, prefer `pwsh` over `powershell`; some environments only expose PowerShell 7.
10. In PowerShell commands, quote peer values such as `"@target_bot"` or `"@username"`.

When this skill is used from another project:

- do not assume the current workspace contains the source repo helpers
- do not require `PATH` if the binary is callable by absolute path
- it is acceptable to identify relevant env-var names such as bot usernames, but never print secret values
- if the user must complete QR or phone+code in a visible terminal, prefer handing them a local `pwsh -File ...` command instead of assuming the agent can surface a new window

## Windows + Git Bash Caveat

On Windows, Git Bash / MSYS2 can rewrite any CLI argument that starts with `/` before it reaches `mi-telegram-cli`.

That means a command such as:

```bash
mi-telegram-cli messages send --profile qa-dev --peer "@target_bot" --text "/start <pairingCode>"
```

can arrive at the binary as:

```text
C:/Program Files/Git/start <pairingCode>
```

This is a shell-host issue, not a parsing bug in the CLI itself.

Use one of these workarounds:

1. Prefix the single invocation with `MSYS_NO_PATHCONV=1`.
2. Export `MSYS_NO_PATHCONV=1` at the top of the Bash script that calls the CLI.
3. Use `pwsh` instead of Git Bash on Windows.

If a bot pairing flow behaves as if `/start <token>` never arrived, read the just-sent message back before debugging the bot. The rewritten payload will be visible there.

## Repo Smoke Helpers

This repo ships smoke helpers for both shells:

- Windows / PowerShell:
  - `tmp\smoke-auth.ps1`
  - `tmp\smoke-bot.ps1`
- Linux / Bash:
  - `tmp/smoke-auth.sh`
  - `tmp/smoke-bot.sh`

They all:

1. pull secrets with `mkey`
2. load `infra/.env`
3. build the binary if needed
4. run the documented v1 CLI flow

## Where To Get `MI_TELEGRAM_API_ID` And `MI_TELEGRAM_API_HASH`

These values come from Telegram, not from the bot or from this repo.

One-time bootstrap:

1. Open `https://my.telegram.org`.
2. Sign in with the Telegram account you use for development bootstrap.
3. Open `API development tools`.
4. Create an app if you do not have one yet.
5. Copy the resulting `api_id` and `api_hash`.

Official references:

- `https://core.telegram.org/api/obtaining_api_id`
- `https://my.telegram.org`

Operational notes:

- Treat `api_hash` as a secret.
- Do not commit it to the repo or paste it into shared logs/chat unless strictly necessary.
- The CLI reads these values from the current shell environment; it does not persist them inside the profile storage root.

Example bootstrap:

```powershell
go build -o .\bin\mi-telegram-cli.exe .\cmd\mi-telegram-cli
go test .\...
$env:PATH = "C:\repos\mios\mi-telegram-cli\bin;$env:PATH"
$env:MI_TELEGRAM_API_ID = "<your_api_id>"
$env:MI_TELEGRAM_API_HASH = "<your_api_hash>"
mi-telegram-cli --help
```

Linux bootstrap shape:

```bash
go build -o ./bin/mi-telegram-cli ./cmd/mi-telegram-cli
go test ./...
export PATH="$(pwd)/bin:$PATH"
bash "$HOME/.agents/skills/mi-key-cli/scripts/mkey.sh" pull mi-telegram-cli dev
set -a
source ./infra/.env
set +a
mi-telegram-cli --help
```

## Existing Authorized QA Profile Flow

```powershell
mi-telegram-cli projects bind --root <repo-root> --profile <existing-authorized-qa-profile> --json
mi-telegram-cli projects current --json
mi-telegram-cli auth status --json
mi-telegram-cli dialogs list --json
mi-telegram-cli messages send --peer "@target_bot" --text hola --json
mi-telegram-cli messages send-photo --peer "@target_bot" --file ".\fixtures\sample.jpg" --caption "QA smoke" --json
mi-telegram-cli messages read --peer "@target_bot" --limit 5 --json
mi-telegram-cli messages press-button --peer "@target_bot" --message-id 301 --button-index 0 --json
mi-telegram-cli messages wait --peer "@target_bot" --timeout 30 --json
```

The project owner must provision and authorize the QA profile before this flow. If `projects bind` rejects it, stop; do not create a profile, run login, or bypass the error.

Operator-managed authentication bootstrap (not for agent-driven QA):

```powershell
mi-telegram-cli auth login --profile qa-dev --method qr
mi-telegram-cli auth login --profile qa-dev --method code --phone +549...
```

Scripted first-use flow:

```powershell
pwsh -File .\tmp\smoke-auth.ps1
pwsh -File .\tmp\smoke-bot.ps1 -Peer "@target_bot"
```

```bash
bash ./tmp/smoke-auth.sh
bash ./tmp/smoke-bot.sh --peer "@target_bot"
```

`tmp/smoke-bot.sh` exports `MSYS_NO_PATHCONV=1` internally so Git Bash does not rewrite slash-leading text payloads sent through the helper.

## Safety Defaults

- Use dedicated Telegram test accounts only.
- Never reuse one profile across different test identities.
- Prefer repo scripts or skills to supply dynamic test inputs such as pairing codes.
- Default daemon mode serializes commands per profile with a FIFO queue. Use `--queue-timeout` for bounded waits, and expect `QueueTimeout` if the command never reaches execution.
- When the bot exposes inline buttons, inspect `buttons[]` first and prefer `button-index` for `messages press-button`.
