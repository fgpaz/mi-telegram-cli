# Task T0: Persist Plan Scaffold

## Shared Context
**Goal:** Preparar el directorio de plan persistido para el cambio de adjuntos y botones inline.
**Stack:** PowerShell, Markdown, wiki local.
**Architecture:** Esta tarea no toca runtime; solo asegura el artefacto de planificación requerido por la skill.

## Task Metadata
```yaml
id: T0
depends_on: []
agent_type: ps-worker
files:
  - create: .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones.md
  - create: .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones/T0-setup.md
  - create: .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones/T1-canon-skill-sync.md
  - create: .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones/T2-telegram-adapter.md
  - create: .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones/T3-cli-surface-tests.md
  - create: .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones/T4-format-verify.md
complexity: low
done_when: "Get-Item .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones.md, .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones | Out-Null"
```

## Reference
`C:\Users\fgpaz\.agents\skills\writing-plans\SKILL.md` — follow the save path and companion-folder rules exactly.

## Prompt
Create the `.docs/raw/plans` directory if it does not exist. Persist the main plan file plus one subdocument per task in the companion folder. Do not write temporary artifacts to repo root and do not mix implementation notes into the main plan file.

## Skeleton
```text
.docs/raw/plans/
  2026-04-14-mi-telegram-cli-adjuntos-botones.md
  2026-04-14-mi-telegram-cli-adjuntos-botones/
```

## Verify
`Get-ChildItem .docs/raw/plans/2026-04-14-mi-telegram-cli-adjuntos-botones*` -> files listed

## Commit
`docs(plan): persist adjuntos y botones inline implementation plan`
