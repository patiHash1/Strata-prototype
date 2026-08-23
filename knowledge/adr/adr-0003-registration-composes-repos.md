---
type: ADR
title: ADR-0003 — Registration composes the repositories
description: RegistrationService now composes the org/user/rbac repositories inside a single transaction instead of duplicating their raw SQL.
tags: [adr, registration]
timestamp: 2026-08-23T00:00:00Z
status: accepted
decided: 2026-08-23
superseded-by: ""
related:
  - /adr/adr-0001-ai-inference-seam.md
---

# ADR-0003 — Registration composes the repositories

`RegistrationService.RegisterWithOwner` previously held the pool and wrote four raw `INSERT` statements that duplicated what `orgRepository`, `userRepository`, and `rbacRepository` already owned. We moved the SQL into the repositories by adding transaction-aware variants (`CreateTx`, `AddMemberTx`, `CreateRoleTx`) that share a private `execer` helper satisfied by both `*pgxpool.Pool` and `pgx.Tx`. `RegistrationService` now builds the four structs and composes the `Tx` methods inside a single transaction, keeping its atomic orchestration and its unique-violation → domain-error translation (`translateOrgInsertErr` / `translateUserInsertErr`).

We kept the pool-based `Create`/`AddMember`/`CreateRole` methods for non-transactional callers and left their error contracts unchanged. The `execer` interface is an internal seam, not a public one.

**Consequences**

- The SQL for each entity lives in exactly one place (its repository).
- Registration is pure orchestration: build structs, call four `Tx` methods, commit.
- The `Tx` variants are available to any future multi-step operation that needs atomicity.