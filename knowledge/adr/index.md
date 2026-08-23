# Architecture Decision Records

* [ADR-0001 — AI inference behind a real seam](adr-0001-ai-inference-seam.md) — all AI inference lives behind a single Infer method with typed request/response pairs, selected by config between a heuristic stub and the internal provider.
* [ADR-0002 — Observability split by responsibility](adr-0002-observability-module-split.md) — the SuperAdmin god module was split into four focused modules behind a composition-root container.
* [ADR-0003 — Registration composes the repositories](adr-0003-registration-composes-repos.md) — RegistrationService composes the org/user/rbac repositories inside a single transaction instead of duplicating their raw SQL.
* [ADR-0004 — Remove the ForTest handler aliases](adr-0004-remove-for-test-aliases.md) — the XxxHandlerForTest aliases were deleted; tests now exercise the real HTTP routes through RoutesForTest, minting a JWT where auth is required.