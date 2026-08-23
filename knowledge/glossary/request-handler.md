---
type: Glossary Term
title: Request Handler
description: The HTTP layer that turns an incoming request into a service call and a service result into an HTTP response — identity extraction, body decode, validation, error mapping.
tags: [http]
timestamp: 2026-08-23T00:00:00Z
---

# Request Handler

The HTTP layer that converts an incoming request into a service call and the result back into an HTTP response. Its responsibilities are identity extraction, body decoding, path-parameter parsing, validation, calling a service, and mapping domain errors to HTTP status codes. Handlers are thin by intent: the domain logic lives in the service layer.

_Avoid_: endpoint controller, action