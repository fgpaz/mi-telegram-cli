# mi-telegram-cli

`mi-telegram-cli` is a local, headless Telegram CLI for repeatable end-to-end QA with real Telegram accounts. It is designed to be called from shells, scripts, and agent skills without depending on a Telegram MCP server.

Useful entry points:

- Repo-local skill: [skills/mi-telegram-cli/SKILL.md](skills/mi-telegram-cli/SKILL.md)
- Quickstart: [skills/mi-telegram-cli/references/quickstart.md](skills/mi-telegram-cli/references/quickstart.md)
- Recipes: [skills/mi-telegram-cli/references/recipes.md](skills/mi-telegram-cli/references/recipes.md)
- Canon: [.docs/wiki/](.docs/wiki/)

## Windows + Git Bash Caveat

When you invoke `mi-telegram-cli` from Git Bash / MSYS2 on Windows, arguments that begin with `/` can be rewritten by the shell host before they ever reach the binary.

Example:

```bash
mi-telegram-cli messages send --profile qa-dev --peer "@SomeBot_bot" --text "/start <token>"
```

Git Bash can rewrite that payload to something like:

```text
C:/Program Files/Git/start <token>
```

The CLI will then send that rewritten text to Telegram as-is. Bots that expect `/start <token>` via a prefix match will not see the command and the pairing flow will fail.

This is not a bug in `mi-telegram-cli` argument parsing. It is MSYS path translation performed by the shell host.

Symptoms:

- the bot falls back to generic onboarding instead of the expected `/start <token>` branch
- reading the outgoing message back shows `C:/Program Files/Git/start <token>` instead of `/start <token>`
- server-side logs never register the `/start` command branch

Supported workarounds:

1. Inline for one command:

```bash
MSYS_NO_PATHCONV=1 mi-telegram-cli messages send --profile qa-dev --peer "@SomeBot_bot" --text "/start <token>"
```

2. Export it in Git Bash scripts before calling the CLI:

```bash
export MSYS_NO_PATHCONV=1
```

3. Use `pwsh` instead of Git Bash on Windows. PowerShell does not rewrite slash-leading arguments.

Notes:

- This affects any text payload that must begin with a literal `/`, including `/start`, `/help`, `/pair`, or a POSIX-style path sent verbatim in chat text.
- `tmp/smoke-bot.sh` already exports `MSYS_NO_PATHCONV=1` to protect Git Bash runs from this rewrite.
- Direct human-mode `messages send` calls may print a one-line warning on suspicious rewritten prefixes to help diagnose this quickly.
