---
type: ADR
title: ADR-0002 — Observability split by responsibility
description: The SuperAdmin god module was split into four focused modules (Telemetry, SOCMonitor, Maintenance, ModuleHealthSvc) behind a composition-root container.
tags: [adr, observability]
timestamp: 2026-08-23T00:00:00Z
status: accepted
decided: 2026-08-23
superseded-by: ""
related:
  - /glossary/telemetry.md
  - /adr/adr-0001-ai-inference-seam.md
---

# ADR-0002 — Observability split by responsibility

The `SuperAdminService` was a god module coordinating five unrelated responsibilities behind ~30 methods: telemetry, SOC events, maintenance, CI health, and module-health aggregation. We split it into four focused modules within `internal/services` — `Telemetry`, `SOCMonitor`, `Maintenance`, and `ModuleHealthSvc` — owned by a `SuperAdmin` composition root that manages the shared pool/redis lifecycle and exposes only `PingRedis`.

We kept the split within `internal/services` (rather than a new package) because this was a structural cleanup, not a swap-seam. `ModuleHealthSvc` composes `Telemetry` and `Maintenance` through interfaces (`HTTPMetricsSource`, `MaintenanceStatusSource`) so a second implementation can be introduced later. We dropped the `FanoutSSEForTest` test alias (test through the real `PublishSOCEvent` surface) and folded `SetUserSvc` into `Telemetry`'s constructor, replacing it with an `ActiveUserCounter` dependency.

**Consequences**

- Each module has its own small interface and its own test surface.
- Middleware now consumes `Telemetry` (logging/recovery) and `Maintenance` (partitioned maintenance) directly instead of the whole god object.
- App holds the four modules plus the container for PingRedis.