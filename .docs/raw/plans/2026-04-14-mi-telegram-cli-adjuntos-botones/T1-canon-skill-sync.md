# Task T1: Canon + Skill Sync

## Shared Context
**Goal:** Alinear el canon y la skill con la nueva superficie observable y accionable del CLI.
**Stack:** Markdown canónico, skill repo-local.
**Architecture:** El contrato visible nace en `.docs/wiki/03-09` y en `skills/mi-telegram-cli`; el código debe seguir exactamente esa superficie.

## Task Metadata
```yaml
id: T1
depends_on:
  - T0
agent_type: ps-worker
files:
  - modify: .docs/wiki/01_alcance_funcional.md
  - modify: .docs/wiki/02_arquitectura.md
  - modify: .docs/wiki/03_FL.md
  - modify: .docs/wiki/03_FL/FL-MSG-01.md
  - modify: .docs/wiki/03_FL/FL-MSG-03.md
  - create: .docs/wiki/03_FL/FL-MSG-05.md
  - modify: .docs/wiki/03_FL/FL-SKL-01.md
  - modify: .docs/wiki/04_RF.md
  - modify: .docs/wiki/04_RF/RF-MSG-001.md
  - modify: .docs/wiki/04_RF/RF-MSG-003.md
  - create: .docs/wiki/04_RF/RF-MSG-005.md
  - modify: .docs/wiki/04_RF/RF-SKL-001.md
  - modify: .docs/wiki/05_modelo_datos.md
  - modify: .docs/wiki/06_matriz_pruebas_RF.md
  - modify: .docs/wiki/06_pruebas/TP-MSG.md
  - modify: .docs/wiki/06_pruebas/TP-SKL.md
  - modify: .docs/wiki/07_baseline_tecnica.md
  - modify: .docs/wiki/07_tech/TECH-SKILL-INTEGRATION.md
  - modify: .docs/wiki/09_contratos_tecnicos.md
  - modify: .docs/wiki/09_contratos/CT-CLI-COMMANDS.md
  - modify: skills/mi-telegram-cli/SKILL.md
  - modify: skills/mi-telegram-cli/references/quickstart.md
  - modify: skills/mi-telegram-cli/references/recipes.md
complexity: medium
done_when: "rg -n \"messages press-button|attachments\\[\\]|buttons\\[\\]|RF-MSG-005|FL-MSG-05\" .docs/wiki skills"
```

## Reference
`.docs/wiki/09_contratos/CT-CLI-COMMANDS.md` — add the public command and selector precedence here first.

## Prompt
Update the canon before or in lockstep with code. Document that `messages read` and `messages wait` return enriched `MensajeResumen` objects with observational `attachments[]` and `buttons[]`. Add the new flow and requirement for `messages press-button`, including typed errors and URL-vs-callback behavior. Update the repo-local skill and recipes so agents stop assuming generic taps or `callback_query` commands and instead inspect `buttons[]` and prefer `buttons[].index`.

## Skeleton
```text
FL-MSG-05 -> RF-MSG-005 -> TP-MSG-023..028 -> CT-CLI-COMMANDS
```

## Verify
`rg -n "messages press-button|attachments\\[\\]|buttons\\[\\]|RF-MSG-005|FL-MSG-05" .docs/wiki skills` -> contract references found

## Commit
`docs(cli): sync canon and skill for attachments and inline buttons`
