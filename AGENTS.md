# mi-telegram-cli Agent Policy

## 1. Workflow Rules

For every non-trivial task in this repository:

1. Run `$ps-contexto` first.
2. Run `$brainstorming` once after context load when there is any open product, contract, data, or architecture decision.
3. Treat `.docs/wiki/01-09` as the mandatory canon before implementation.
4. If the task changes documentation policy, update `AGENTS.md` and `CLAUDE.md` together using `$ps-crear-agentsclaudemd`.
5. Close non-trivial tasks with `$ps-trazabilidad`.

Additional strict rules:

- Use `crear-alcance`, `crear-arquitectura`, `crear-flujo`, `crear-modelo-de-datos`, `crear-capa-tecnica-wiki`, and `crear-requerimiento` for their matching layers only.
- Do not implement code that changes visible behavior, CLI contracts, states, or storage assumptions without reviewing the affected docs in `.docs/wiki/01-09`.
- If future work introduces UX/UI canon, use the hardened numbered canon directly; if legacy UX/UI paths ever appear, stop and run `$ps-migrar-canon-uxui` before downstream UX/UI work.

## 2. Canonical Source of Truth

Documentation root:

- `.docs/wiki/`

Functional truth:

- `.docs/wiki/01_alcance_funcional.md`
- `.docs/wiki/02_arquitectura.md`
- `.docs/wiki/03_FL.md`
- `.docs/wiki/03_FL/`
- `.docs/wiki/04_RF.md`
- `.docs/wiki/04_RF/`
- `.docs/wiki/05_modelo_datos.md`
- `.docs/wiki/06_matriz_pruebas_RF.md`
- `.docs/wiki/06_pruebas/`

Technical truth:

- `.docs/wiki/07_baseline_tecnica.md`
- `.docs/wiki/07_tech/`
- `.docs/wiki/08_modelo_fisico_datos.md`
- `.docs/wiki/08_db/`
- `.docs/wiki/09_contratos_tecnicos.md`
- `.docs/wiki/09_contratos/`

Skill source of truth:

- `skills/mi-telegram-cli/`

Policy docs:

- `AGENTS.md`
- `CLAUDE.md`

## 3. Skill Distribution Policy

The canonical editable skill lives in:

- `skills/mi-telegram-cli/`

The global Codex install lives in:

- `C:\Users\fgpaz\.codex\skills\mi-telegram-cli`

The external mirror lives in:

- `C:\repos\buho\assets\skills\mi-telegram-cli`

The mirrored binary lives in:

- `C:\repos\buho\assets\skills\mi-telegram-cli\bin\mi-telegram-cli.exe`

Mandatory rule:

- If the repo-local skill changes and the task also updates or reinstalls the global copy under `C:\Users\fgpaz\.codex\skills\mi-telegram-cli`, update the mirror under `C:\repos\buho\assets\skills\mi-telegram-cli` in the same task.
- If the task rebuilds or redistributes `mi-telegram-cli.exe`, update the mirrored binary under `C:\repos\buho\assets\skills\mi-telegram-cli\bin\mi-telegram-cli.exe` in the same task.
- Do not leave drift between the repo skill, the global installed copy, and the `C:\repos\buho\assets\skills` mirror once a global update happened.
- If updating the mirror requires permissions outside the current workspace, request approval instead of silently skipping the sync.

## 4. Documentation Synchronization Rule

When product objective, scope boundary, or MVP limits change:

- review/update `.docs/wiki/01_alcance_funcional.md`

When architecture, decision priority, runtime model, or system boundaries change:

- review/update `.docs/wiki/02_arquitectura.md`
- review/update `.docs/wiki/07_baseline_tecnica.md`
- review/update affected `TECH-*`

When actors, flow ownership, sequence, or risk mitigation change:

- review/update `.docs/wiki/03_FL.md`
- review/update affected `FL-*`

When CLI commands, typed inputs/outputs, errors, or smoke behavior change:

- review/update `.docs/wiki/04_RF.md`
- review/update affected `RF-*`
- review/update `.docs/wiki/06_matriz_pruebas_RF.md`
- review/update affected `TP-*`
- review/update `.docs/wiki/09_contratos_tecnicos.md`
- review/update affected `CT-*`

When entities, invariants, persisted-vs-derived decisions, or lifecycle rules change:

- review/update `.docs/wiki/05_modelo_datos.md`
- review/update `.docs/wiki/08_modelo_fisico_datos.md`
- review/update affected `DB-*`

## 5. Artifact Hygiene Rule

- Do not write ephemeral artifacts to repository root.
- Do not leave screenshots, logs, traces, auth dumps, browser captures, or temporary exports under `.docs/wiki/` or `skills/`.
- Use `tmp/` for disposable local scratch output when a tool needs a working directory.
- Use `artifacts/e2e/<YYYY-MM-DD>-<task-slug>/` for task-scoped E2E evidence when evidence must be kept in the repo.
- Prefer explicit output paths for scripts and runners whenever configurable.
- Before closing a task, clean, relocate, or delete ephemeral outputs that are not part of the canon.

## 6. Placeholder Mapping

- `<DOCS_ROOT>` -> `.docs/wiki/`
- `<ARQUITECTURA_DOC>` -> `.docs/wiki/02_arquitectura.md`
- `<FL_INDEX_DOC>` -> `.docs/wiki/03_FL.md`
- `<FL_DOCS_DIR>` -> `.docs/wiki/03_FL/`
- `<RF_INDEX_DOC>` -> `.docs/wiki/04_RF.md`
- `<RF_DOCS_DIR>` -> `.docs/wiki/04_RF/`
- `<MODELO_DATOS_DOC>` -> `.docs/wiki/05_modelo_datos.md`
- `<TP_INDEX_DOC>` -> `.docs/wiki/06_matriz_pruebas_RF.md`
- `<TP_DOCS_DIR>` -> `.docs/wiki/06_pruebas/`
- `<BASELINE_TECNICA_DOC>` -> `.docs/wiki/07_baseline_tecnica.md`
- `<MODELO_FISICO_DOC>` -> `.docs/wiki/08_modelo_fisico_datos.md`
- `<CONTRATOS_TECNICOS_DOC>` -> `.docs/wiki/09_contratos_tecnicos.md`
- `<TECH_DOCS_DIR>` -> `.docs/wiki/07_tech/`
- `<DB_DOCS_DIR>` -> `.docs/wiki/08_db/`
- `<CONTRATOS_DOCS_DIR>` -> `.docs/wiki/09_contratos/`
- `<SKILL_SOURCE_DIR>` -> `skills/mi-telegram-cli/`
- `<GLOBAL_SKILL_DIR>` -> `C:\Users\fgpaz\.codex\skills\mi-telegram-cli`
- `<SKILL_MIRROR_DIR>` -> `C:\repos\buho\assets\skills\mi-telegram-cli`
- `<SKILL_MIRROR_BIN>` -> `C:\repos\buho\assets\skills\mi-telegram-cli\bin\mi-telegram-cli.exe`

## 7. Search Playbook

Use fast discovery first:

```powershell
rg --files .docs/wiki
rg -n "FL-|RF-|TP-|TECH-|DB-|CT-|profile|auth|dialog|message|skill" .docs/wiki skills
rg -n "mi-telegram-cli|profiles|auth|dialogs|messages|WaitTimeout|PeerAmbiguous" .docs/wiki skills
```

## 8. Non-Negotiables

- Keep `AGENTS.md` and `CLAUDE.md` aligned.
- Do not change the global skill without considering the `C:\repos\buho\assets\skills` mirror.
- Do not implement code first and retrofit `01-09` later for behavior-changing work.
- Do not treat generated local state or Telegram session blobs as canonical documentation.
