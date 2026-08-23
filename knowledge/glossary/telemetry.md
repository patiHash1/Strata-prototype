---
type: Glossary Term
title: Telemetry
description: The measured runtime health of the platform — runtime memory/GC, database pool state, HTTP request counts, and latency percentiles.
tags: [observability]
timestamp: 2026-08-23T00:00:00Z
related:
  - /adr/adr-0002-observability-module-split.md
---

# Telemetry

The measured runtime health of the platform: Go runtime memory and GC stats, Postgres connection-pool state, HTTP request volume by status, and latency percentiles. It is distinct from SOC (security events) and from module health scores — telemetry is the raw signal; health scores are a composite judgement derived from it.

_Avoid_: metrics grab-bag, monitoring