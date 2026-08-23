---
type: ADR
title: ADR-0001 — AI inference behind a real seam
description: All AI inference lives behind a single Infer method with typed request/response pairs, selected by config between a heuristic stub and the internal provider.
tags: [adr, ai]
timestamp: 2026-08-23T00:00:00Z
status: accepted
decided: 2026-08-23
superseded-by: ""
related:
  - /glossary/ai-inference.md
---

# ADR-0001 — AI inference behind a real seam

The five domain modules (CRM, Accounting, SupplyChain, HR, Platform) each contained their own shallow AI heuristic functions calling `rand` directly. We consolidated all genuine inference into a single `internal/ai` module behind one `Infer(ctx, Request) (Response, error)` method, with typed request/response pairs. The seam is real: the heuristic stub and the internal provider are two adapters, selected by the `AI_PROVIDER` config value (default `stub`).

We chose a single `Infer` method over per-capability methods to keep the interface maximally deep, and typed request/response over an opaque string-keyed blob so call sites stay type-safe. The `ai` package is dependency-free (no import of `services`) to avoid a circular import; the four request types that reference domain types use `ai`-defined value types, translated at the call site. Deterministic derivations (chart recommendation, severity matching, geofence) stay in their domain modules; tightly-coupled ones (priority, reorder quantity, risk rating) are folded into the responses.

**Considered Options**

- **Per-capability methods** — rejected: interface nearly as wide as the implementation (shallow).
- **Generic string-keyed blob** — rejected: small interface but an opaque protocol that leaks across the seam.

**Consequences**

- The provider adapter is a placeholder returning `ErrInference` until the internal provider lands.
- The interface is the test surface: swap the adapter at construction to test the seam.