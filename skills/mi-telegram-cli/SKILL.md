---
name: mi-telegram-cli
description: "Use when Codex needs to drive local Telegram end-user QA through the `mi-telegram-cli` binary: authenticate a dedicated Telegram account, manage isolated profiles, inspect enriched messages and adjuntos, resolve peers, send text messages, press inline buttons, wait for replies, mark chats as read, or run reproducible E2E bot smokes without relying on an MCP server."
---

# mi-telegram-cli

Use this skill to operate Telegram locally through the `mi-telegram-cli` binary with isolated profiles and dedicated test accounts.

Prefer the CLI over any Telegram MCP when the goal is local control of sessions, multicuenta isolation, and reproducible bot QA.

## Operating Rules

- Keep one profile per Telegram test account and environment.
- Use dedicated QA accounts only, never personal accounts.
- Keep Telegram logic in the CLI; the skill should only orchestrate shell commands.
- Do not assume the current workspace is the `mi-telegram-cli` source repo. This skill must work from other projects too.
- Never run commands concurrently against the same profile. Even `auth status`, `me`, `dialogs list`, or `messages read` can hit `ProfileLocked` while another operation is still active.
- If `mi-telegram-cli` is not on `PATH`, that is not a blocker by itself. First try a known absolute path or bootstrap the binary from the source repo before considering MCP fallback.
- If the binary is missing, bootstrap it from this repo instead of falling back to an MCP, unless the user explicitly asks for MCP fallback.
- On Windows, prefer `pwsh` over `powershell`. Some environments expose only PowerShell 7, and `powershell.exe` may be absent from `PATH`.
- In PowerShell examples, always quote peer values such as `"@multi_tedi_dev_bot"` or `"@username"`. Unquoted `@...` can be parsed incorrectly before reaching the CLI.
- If `MI_TELEGRAM_API_ID` or `MI_TELEGRAM_API_HASH` are missing, explain that they come from creating a Telegram app in `https://my.telegram.org` under `API development tools`.
- If the user asks for the current official steps to obtain those values, consult Telegram's official sources dynamically instead of guessing:
  - `https://core.telegram.org/api/obtaining_api_id`
  - `https://my.telegram.org`
- Never ask the user to paste `api_hash` into chat unless strictly necessary; treat it as a secret.
- When an interactive login window must be visible to the user, prefer giving the user an explicit local `pwsh -File <script>` or direct `auth login` command. Agent-launched windows may not surface on the user's desktop in every host/session setup.
- The `tmp/smoke-*` helpers are conveniences from the `mi-telegram-cli` source repo. If they are not present in the current workspace, that is expected in consumer repos; drive the installed binary directly instead of treating their absence as a problem.
- Only inspect the current project's docs or code when the user needs help identifying the target bot, peer, pairing flow, or Telegram-triggered behavior under test.
- It is fine to identify the names of relevant env vars or config keys in a consumer repo, but do not print or paraphrase their secret values.

## Quick Start

1. Verify whether the binary exists and is callable.
2. If it is not on `PATH`, try the known absolute binary path or a source-repo `bin/` path before treating it as missing.
3. If the binary is still missing, follow [references/quickstart.md](references/quickstart.md) to install or build it.
4. Create or reuse a profile.
5. Login the dedicated account.
   In a real interactive terminal, `auth login --profile <id>` can omit `--method` and the CLI will ask `QR` or `Phone + code`.
   For scripts, agents, and smokes, keep `--method code` or `--method qr` explicit.
   On Windows, prefer `pwsh` for helper scripts and local interactive wrappers.
6. Resolve the target bot or dialog.
   In PowerShell, quote peer values such as `"@target_bot"`.
7. Send text, inspect `attachments[]` / `buttons[]`, press inline buttons when needed, wait for reply, and optionally mark the chat as read.
   Keep each profile's commands serial; do not parallelize reads or status checks on the same profile.

Repo-local smoke helpers exist for both Windows and Linux:

- PowerShell: `tmp/smoke-auth.ps1`, `tmp/smoke-bot.ps1`
- Bash/Linux: `tmp/smoke-auth.sh`, `tmp/smoke-bot.sh`

Use them when the user wants a repeatable local auth/bootstrap flow or a deterministic bot smoke without retyping the CLI sequence by hand.
If the current workspace is not the `mi-telegram-cli` source repo and those scripts are missing, skip them and use direct `mi-telegram-cli` commands.

## Canonical Command Surface

- `profiles add|list|show|remove`
- `auth login|status|logout`
- `me`
- `dialogs list|mark-read`
- `messages read|send|wait|press-button`

Use `--json` for any agent-driven flow except `auth login --method qr`, which is terminal-interactive by design.
Even though humans may omit `--method` in an interactive TTY, agent-driven flows should keep `--method` explicit.

## Common Recipes

- For installation/bootstrap and current repo status, read [references/quickstart.md](references/quickstart.md).
- For the `multi-tedi` smoke sequence, cross-account messaging, and peer-handling patterns, read [references/recipes.md](references/recipes.md).
- For environment bootstrap and where to obtain Telegram API credentials, read [references/quickstart.md](references/quickstart.md).
- For repo-provided Windows/Linux smoke scripts, read [references/recipes.md](references/recipes.md).

When this skill is used inside another project:

- do not assume that project's workspace contains `tmp/smoke-*`
- do not read its wiki by default just because the skill was invoked
- inspect that project's docs only when needed to find the correct bot, peer, pairing code, or trigger path

## Output Discipline

- Prefer `--json` and summarize only the fields needed for the next step.
- Use `buttons[].index` as the canonical selector when the flow requires `messages press-button`; use `button-text` only as fallback.
- Treat `attachments[]` and `buttons[]` as observational metadata; this skill should not assume downloads or generic UI taps exist unless the CLI explicitly exposes them.
- Treat `WaitTimeout`, `PeerAmbiguous`, and `UnauthorizedProfile` as first-class failures.
- Do not persist or echo secrets, auth codes, or session blobs in chat output.

## Shell Caveats (Git Bash on Windows)

When invoking `mi-telegram-cli` from Git Bash / MSYS2 on Windows, arguments that begin with `/` are automatically rewritten by the MSYS path-translation layer. A text like `/start <token>` becomes `C:/Program Files/Git/start <token>` before it reaches the CLI. The CLI then sends that rewritten string to Telegram as-is, and any bot that matches `/start` via `StartsWith("/start ")` will never see the command.

Symptoms that diagnose this issue:

- The bot replies with its unlinked-guidance / generic onboarding message for any `/start <token>`.
- Reading the outgoing message back through `messages read` shows the rewritten prefix, for example `C:/Program Files/Git/start <token>`, instead of `/start <token>`.
- Server-side webhook logs never register the `/start` command branch for the attempted pairing.

Three supported workarounds:

1. `MSYS_NO_PATHCONV=1 mi-telegram-cli messages send --profile <id> --peer "@bot" --text "/start <token>"` - disables MSYS path translation for a single invocation. Preferred for ad-hoc tests.
2. Use PowerShell (`pwsh`) instead of Git Bash; PowerShell does not rewrite arguments.
3. In scripts, export `MSYS_NO_PATHCONV=1` at the top so every call inherits the exemption.

This affects any text payload that should reach Telegram starting with a literal `/`: bot commands (`/start`, `/help`, `/pair`), slash-prefixed flags the user wants to send verbatim, and POSIX-style paths inside chat text. It does not affect `--peer` values, `--profile` ids, or non-leading slashes (`foo/bar`).

When reproducing a bot's command flow from Git Bash, always verify the outgoing text via `messages read ... --json` before blaming the bot or the webhook - the rewrite happens client-side and is invisible until you round-trip the payload.
