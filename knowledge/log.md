## 2026-08-23

**Creation** — Recorded [ADR-0003 — Registration composes the repositories](/knowledge/adr/adr-0003-registration-composes-repos.md).

**Backlog cleared** — Migrated the remaining handler inline boilerplate to the `request_helpers.go` helpers across `handlers_{accounting,accounting_extra,billing,crm,hr,hr_extra,org,platform,platform_extra,supplychain,supplychain_extra,super_admin,auth}.go`. Remaining non-helper cases are intentional: API-key claim handlers (`GetAPIKeyClaims`), the optional-claims logout handler, and non-UUID path params.

**Update** — Added glossary term [Request Handler](/knowledge/glossary/request-handler.md).

**Backlog** — Extracted request-handler helpers in `internal/handlers/request_helpers.go` and migrated a representative subset (account, org members, accounting journal entry). Remaining handlers still use the inline pattern and should migrate to the helpers incrementally — tracked as a follow-up.

## 2026-08-23

**Update** — Recorded [ADR-0002 — Observability split by responsibility](/knowledge/adr/adr-0002-observability-module-split.md); added glossary term [Telemetry](/knowledge/glossary/telemetry.md).

## 2026-08-23

**Creation** — Added glossary term [AI Inference](/knowledge/glossary/ai-inference.md).
**Creation** — Recorded [ADR-0001 — AI inference behind a real seam](/knowledge/adr/adr-0001-ai-inference-seam.md).