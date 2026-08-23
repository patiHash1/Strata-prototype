---
type: ADR
title: ADR-0004 — Remove the ForTest handler aliases
description: The XxxHandlerForTest aliases were deleted; tests now exercise the real HTTP routes through RoutesForTest, minting a JWT where auth is required.
tags: [adr, testing]
timestamp: 2026-08-23T00:00:00Z
status: accepted
decided: 2026-08-23
superseded-by: ""
related:
  - /adr/adr-0003-registration-composes-repos.md
---

# ADR-0004 — Remove the ForTest handler aliases

The super-admin handlers exposed ~17 exported `XxxHandlerForTest` methods that simply called the real unexported handler, so tests could hit a single handler on a bare mux and bypass the auth middleware. These were pass-throughs — the deletion test says deleting them moves nothing. We deleted them and migrated the tests to exercise the real HTTP routes through `RoutesForTest()`, minting a valid JWT (`AuthService.CreateToken` with `PermSuperAdmin`) where a route requires auth.

`RoutesForTest()` remains as the single legitimate test seam. The `{module}/health` route is public and needs no token. For `RequireAuthCookie` routes, tests wait for the next wall-clock second before minting, because the middleware rejects tokens whose `IssuedAt` is not strictly after the app's `startedAt`.

**Consequences**

- Tests now cross the real interface (the HTTP route + middleware stack), not a fake seam.
- The interface is the test surface: a change to auth or routing is now caught by the tests.
- `RoutesForTest()` is the only test-only export left in the handlers package.