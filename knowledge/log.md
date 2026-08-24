## 2026-08-24

**Decision** — Architecture review (report: `tmp/architecture-review-20260824.html`) resolved to work through all six candidates sequentially, starting with #1.

**Decision** — Candidate #1 (give the domain services a real seam): **pilot on CRM only**, then replicate. The seam shape (repo interface, in-memory adapter, error mapping) is learned on CRM; the other four domains — Accounting, SupplyChain, HR, Platform (`services_{accounting,supplychain,hr,platform}.go`) — are near-mechanical replication of CRM's settled shape. Replication checklist when resuming:

1. Extract the concrete repo into a **consumer-side, minimal interface** declared in `internal/services` — the service declares only the methods its implementation uses; the pgx repo satisfies it implicitly (decided 2026-08-24).
2. Constructor keeps taking `*pgxpool.Pool` as prod default; injection via **functional options** (`WithCRMRepo(...)` etc.) so future dependencies don't change existing call sites (decided 2026-08-24).
3. In-memory adapter per repo in a dedicated package (e.g. `internal/services/crmmem`) so handler tests can reuse it; the adapter reproduces **error modes** (`ErrNotFound`, not-in-org, duplicate → domain errors) since they're contractual, but not SQL semantics — no testcontainers for this prototype (decided 2026-08-24).
4. Watch for transaction-aware variants (ADR-0003's `execer` pattern) where repos compose in transactions.
5. Pilot tests: CRUD round-trips + error modes **and** one composition path (`CreateLead` with a scripted `ai.Inferrer` fake; exact score assertions). Nondeterminism absorbed at the seam: inject a `WinProbability` scorer dependency via functional option, defaulting to the current `rand.Intn` behaviour (decided 2026-08-24). **Done:** expanded to full coverage of every CRM method (see below).

**Deepening** — Candidate #1 (CRM real seam) implemented and fully covered. Added `CRMRepository` consumer-side interface, `CRMOption`/`WithCRMRepo`/`WithWinProbability` functional options, and `internal/services/crmmem` in-memory adapter (`SeedQuote`/`SeedCampaign`/`SeedContact`, `AllDeals`/`AllVisits`/`AllCampaigns`, `FailNext` error-mode injection). Tests in `internal/services/services_crm_test.go` now cover every CRM service method: `CreateLead` (win-probability pinning + deal-title/amount variants), `AnalyzeContractRisk` (happy + ErrQuoteNotFound + ErrQuoteNotInOrg + persistence), `CreateTicket` (happy + ErrContactNotFound + ErrContactNotInOrg + repo failure), `ScheduleFieldVisit` (happy + repo failure), `CreateCampaign` (segment-criteria persistence + repo failure), `LaunchCampaign` (happy + ErrCampaignNotFound + ErrCampaignNotInOrg), plus repo-failure surfacing. `go build ./...`, `go vet`, and `go test ./...` all pass.

Follow-up candidates queued after #1: #2 error-mapping mapper (`Strong`), #3 API-key auth out of SupplyChain, #4 UserService relay deletion, #5 super-admin handler split, #6 typed wrappers over `ai.Infer` (speculative).

**Deepening** — Candidate #2 (collapse the error-mapping ladder) implemented. Added `writeServiceErr(w, err, fallbackMsg)` and a `serviceError` sentinel→(status, message) table in `internal/handlers/request_helpers.go`; empty message means "use err.Error()". Migrated all 9 handler sites across `handlers_{crm,accounting,auth,hr,supplychain,account}.go` from hand-rolled `errors.Is`/`switch` ladders to a single `if writeServiceErr(...) { return }` line. Added `internal/handlers/request_helpers_test.go` covering sentinel mapping (custom + err.Error messages) and the unknown-error 500 fallback. `go build`, `go vet`, `go test ./...` all pass.

**Candidate** — Candidate #3 (API-key auth out of SupplyChain): decided new module lives in `internal/services/apikey` with an `APIKeyService` (mirrors `crmmem` precedent, keeps auth in services layer, avoids bloating `AuthService`) (decided 2026-08-24).

**Decision** — Candidate #3: `APIKeyService` accepts a consumer-side minimal `APIKeyRepository` seam (prefix lookup) + injected bcrypt verify; `apiKeyRecord` moves to `internal/services/apikey` and becomes exported (e.g. `APIKeyRecord`); `ValidateAPIKey` keeps the `(uuid.UUID, []string, error)` signature so `utils.RequireAPIKey`'s structural interface keeps working unchanged (decided 2026-08-24).

**Decision** — Candidate #3 service shape: `NewAPIKeyService(pool *pgxpool.Pool, verify VerifyFunc, opts...)` where `verify` is a bare `func(hash, password string) bool` defaulting to bcrypt (NOT `*AuthService`); `WithAPIKeyRepo(repo)` functional option substitutes the in-memory adapter for tests (decided 2026-08-24).

**Decision** — Candidate #3 wiring: construct `*apikey.APIKeyService` in `main.go` (passing the pgx pool + bcrypt verify), thread through `handlers.New` onto `App`, and use it in `RequireAPIKey` at the route (decided 2026-08-24).

**Decision** — Candidate #3 tests: hand-rolled fake `APIKeyRepository` in the test file (no separate mem package — only the service's own tests use it, unlike `crmmem`). Destination module takes only `GetAPIKeyByPrefix`; the dead `GetActiveAPIKeyRecords` is deleted with the move, and `extractAPIKeyPrefix` moves into the new module as the service's helper (decided 2026-08-24).

**Deepening** — Candidate #3 implemented. New module `internal/services/apikey` (`APIKeyService`, `APIKeyRepository` consumer-side seam, `WithRepo`/`WithVerify` options, `extractAPIKeyPrefix`, pgx `repo`, `ErrInvalid`). Removed from `SupplyChainService`: `authSvc` dep, `ValidateAPIKey`, `extractAPIKeyPrefix`, `apiKeyRecord`, both API-key repo methods (incl. dead `GetActiveAPIKeyRecords`), and `ErrAPIKeyInvalid`. `NewSupplyChainService(pool, ai)` no longer takes auth. Wired `*apikey.APIKeyService` via `main.go` → `handlers.New` → `App.APIKey`; route now `utils.RequireAPIKey(a.APIKey, ...)`. `utils.RequireAPIKey`'s structural interface unchanged. Tests in `apikey_test.go` (fake repo): invalid format, unknown prefix, bad hash, success, nil-scopes normalization. `go build`, `go vet`, `go test ./...` pass.

**Candidate** — Candidate #4 (UserService relay): decided to **deepen** (not delete). The membership/role validation currently scattered across `handlers_org.go` (`updateMemberHandler`/`deleteMemberHandler`/`removeMemberHandler` each re-check `GetMemberByID` + OrgID) moves **into** `UserService` so the relays earn their layer and handlers shrink; tests cross one interface (decided 2026-08-24).

**Decision** — Candidate #4 scope: deepens only the three write methods (`UpdateMemberRole`, `DeactivateMember`, `RemoveMember`) to add `(orgID, actingUserID)` params + internal not-found/not-in-org/self-targeting guards returning sentinel errors; deletes unused `ListMembersByUserPage` and the now-unused `GetMemberByID` service relay (repo method stays, used internally); **keeps** `AddMember`/`GetMember` as-is because `SeedService.SeedSuperAdmin` uses them — touching them would ripple into the seed/registration transaction flow, tracked as backlog (decided 2026-08-24).

**Deepening** — Candidate #4 implemented. `UserService` gained a consumer-side `UserRepository` seam + `WithUserRepo` option (mirrors candidate #1); `CountActiveUsers`/`UpdateLastLoginAt` moved from inline `s.repo.pool` SQL into repo methods so the interface is satisfied. Deepened `UpdateMemberRole`/`DeactivateMember`/`RemoveMember` to `(ctx, orgID, actingUserID, memberID, ...)` with a shared `requireMemberForOrg` guard (exists, in-org, not-self) returning new sentinels `ErrMemberNotFound`/`ErrMemberNotInOrg`/`ErrSelfChange`/`ErrMemberAlreadyDeactivated`. Deleted dead `ListMembersByUserPage` and `GetMemberByID` service relays. The three org handlers (`updateMemberHandler`/`deleteMemberHandler`/`removeMemberHandler`) dropped their duplicated `GetMemberByID`+nil+org+self guards to a single `writeServiceErr` call. New sentinels registered in the `writeServiceErr` mapper. Tests in `services_users_test.go` (hand-rolled fake repo): guard coverage for all three methods. `go build`, `go vet`, `go test ./...` pass.

**Backlog** — Candidate #4 follow-up: `AddMember`/`GetMember` remain shallow relays used only by `SeedService.SeedSuperAdmin`; revisit whether to deepen or expose the repo to the seed flow (decided 2026-08-24).

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