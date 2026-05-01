# Gobierno documental

```yaml
doc_id: DOC-GOV-MI-TELEGRAM-CLI
profile: spec_backend
governance_profile: spec_backend
docs_root: .docs/wiki
projection:
  output: .docs/wiki/_mi-lsp/read-model.toml
  format: toml
  auto_sync: true
  versioned: true
canonical_layers:
  scope: 01_alcance_funcional.md
  architecture: 02_arquitectura.md
  flows: 03_FL.md
  requirements: 04_RF.md
  data_model: 05_modelo_datos.md
  tests: 06_matriz_pruebas_RF.md
  technical_baseline: 07_baseline_tecnica.md
  physical_model: 08_modelo_fisico_datos.md
  technical_contracts: 09_contratos_tecnicos.md
owners:
  cli_runtime: cmd/mi-telegram-cli
  app_runtime: internal/app
  profile_storage: internal/profile
  telegram_adapter: internal/tg
  skill_source: skills/mi-telegram-cli
verify:
  - go test ./...
stop_if:
  - canon 01-09 contradicts CLI behavior
  - profile/session/audit artifacts expose Telegram secrets or message bodies
evidence:
  durable: artifacts/e2e/ or .docs/wiki/contexto/
hierarchy:
  - id: governance
    label: Gobierno documental
    layer: "00"
    family: functional
    paths:
      - .docs/wiki/00_gobierno_documental.md
  - id: scope
    label: Alcance funcional
    layer: "01"
    family: functional
    paths:
      - .docs/wiki/01_alcance_funcional.md
  - id: architecture
    label: Arquitectura
    layer: "02"
    family: functional
    paths:
      - .docs/wiki/02_arquitectura.md
  - id: flow
    label: Flujos
    layer: "03"
    family: functional
    paths:
      - .docs/wiki/03_FL.md
      - .docs/wiki/03_FL/*.md
  - id: requirements
    label: Requerimientos funcionales
    layer: "04"
    family: functional
    paths:
      - .docs/wiki/04_RF.md
      - .docs/wiki/04_RF/*.md
  - id: semantic_data
    label: Modelo de datos
    layer: "05"
    family: data
    paths:
      - .docs/wiki/05_modelo_datos.md
  - id: tests
    label: Pruebas
    layer: "06"
    family: qa
    paths:
      - .docs/wiki/06_matriz_pruebas_RF.md
      - .docs/wiki/06_pruebas/*.md
  - id: technical_baseline
    label: Baseline tecnica
    layer: "07"
    family: technical
    paths:
      - .docs/wiki/07_baseline_tecnica.md
      - .docs/wiki/07_tech/*.md
  - id: physical_data
    label: Modelo fisico
    layer: "08"
    family: technical
    paths:
      - .docs/wiki/08_modelo_fisico_datos.md
      - .docs/wiki/08_db/*.md
  - id: contracts
    label: Contratos tecnicos
    layer: "09"
    family: technical
    paths:
      - .docs/wiki/09_contratos_tecnicos.md
      - .docs/wiki/09_contratos/*.md
context_chain:
  - governance
  - scope
  - architecture
  - flow
  - requirements
  - technical_baseline
  - contracts
closure_chain:
  - governance
  - flow
  - requirements
  - technical_baseline
  - contracts
  - tests
audit_chain:
  - governance
  - flow
  - requirements
  - technical_baseline
  - physical_data
  - contracts
  - tests
blocking_rules:
  - missing_governance
  - stale_projection
  - canon_contract_drift
  - secret_leakage
```

Este documento declara la autoridad mínima del wiki para `mi-telegram-cli`.
Los cambios que afecten comportamiento visible, storage, contratos CLI, daemon, auditoría o distribución de skill deben mantener sincronizados `01-09`, `AGENTS.md` y `CLAUDE.md` cuando corresponda.
