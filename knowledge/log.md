## 2026-08-23

**Evaluated** — Candidate 5 (deepen the HR module): the HR module's resume parsing, match scoring, relevance, and shift prediction already flow through the `Inferrer` seam from candidate 1's cleanup; the remaining `simulateGeofence` is a deterministic derivation that per ADR-0001 correctly stays local to `services_hr.go`. No code change needed.

**Evaluated** — Candidate 6 (deepen the Accounting module): the module's invoice OCR and expense-fraud audit already flow through the `Inferrer` seam (`ai.OCRRequest`/`ai.ExpenseAuditRequest`) from candidate 1's cleanup; the only remaining `rand` call is entry-number generation in `PostJournalEntry`, which is serial/ID generation, not AI inference. No code change needed.

**Evaluated** — Candidate 7 (deepen the SuperAdmin composition root): the root is already the desired deep shape — handlers depend on the four concrete observability modules directly (`a.Telemetry`/`a.SOCMonitor`/`a.Maintenance`/`a.ModuleHealth`), and `SuperAdmin` holds only lifecycle (pool/redis, `PingRedis`, `Shutdown`). No pass-through remains. No code change needed.

**Deepening** — Routed the Platform module's IoT reading anomaly detection through the [AI Inference seam](/knowledge/glossary/ai-inference.md): added `ReadingAnomalyRequest`/`ReadingAnomalyResponse` typed pair and its stub adapter, and deleted the local `DetectReadingAnomaly` helper that had bypassed the seam.

**Deepening** — Routed the CRM campaign module's audience segmentation and reach estimation through the [AI Inference seam](/knowledge/glossary/ai-inference.md): added `CampaignSegmentRequest`/`CampaignReachRequest` typed pairs and their stub adapters, and deleted the local `aiSegmentAudience`/`aiEstimateReach` helpers that had bypassed the seam.

**Deepening** — Routed the SupplyChain module's work-order bottleneck risk and purchase-order supplier risk rating through the [AI Inference seam](/knowledge/glossary/ai-inference.md): added `BottleneckRiskRequest`/`SupplierRiskRatingRequest` typed pairs and their stub adapters, and deleted the local `aiPredictBottleneckRisk`/`aiPredictSupplierRisk`/`aiSupplierRiskRating` helpers that had bypassed the seam.

**Cleanup** — Removed ~20 dead `ai*`/`simulate*` heuristic helpers from the five domain service files (`services_crm.go`, `services_accounting.go`, `services_hr.go`, `services_platform.go`, `services_supplychain.go`). They were duplicate, unreferenced implementations of the AI Inference capability that bypassed the [Inference seam](/knowledge/glossary/ai-inference.md); all genuine inference already flows through the single `Inferrer` interface per ADR-0001. Imports left unused by the deletions were pruned.

**Creation** — Recorded [ADR-0004 — Remove the ForTest handler aliases](/knowledge/adr/adr-0004-remove-for-test-aliases.md).

**Creation** — Recorded [ADR-0003 — Registration composes the repositories](/knowledge/adr/adr-0003-registration-composes-repos.md).

**Backlog cleared** — Migrated the remaining handler inline boilerplate to the `request_helpers.go` helpers across `handlers_{accounting,accounting_extra,billing,crm,hr,hr_extra,org,platform,platform_extra,supplychain,supplychain_extra,super_admin,auth}.go`. Remaining non-helper cases are intentional: API-key claim handlers (`GetAPIKeyClaims`), the optional-claims logout handler, and non-UUID path params.

**Update** — Added glossary term [Request Handler](/knowledge/glossary/request-handler.md).

**Backlog** — Extracted request-handler helpers in `internal/handlers/request_helpers.go` and migrated a representative subset (account, org members, accounting journal entry). Remaining handlers still use the inline pattern and should migrate to the helpers incrementally — tracked as a follow-up.

## 2026-08-23

**Update** — Recorded [ADR-0002 — Observability split by responsibility](/knowledge/adr/adr-0002-observability-module-split.md); added glossary term [Telemetry](/knowledge/glossary/telemetry.md).

## 2026-08-23

**Creation** — Added glossary term [AI Inference](/knowledge/glossary/ai-inference.md).
**Creation** — Recorded [ADR-0001 — AI inference behind a real seam](/knowledge/adr/adr-0001-ai-inference-seam.md).