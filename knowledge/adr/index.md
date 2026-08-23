# Architecture Decision Records

* [ADR-0001 — AI inference behind a real seam](adr-0001-ai-inference-seam.md) — all AI inference lives behind a single Infer method with typed request/response pairs, selected by config between a heuristic stub and the internal provider.