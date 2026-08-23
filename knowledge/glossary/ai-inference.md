---
type: Glossary Term
title: AI Inference
description: The capability that turns unstructured input (ticket text, contracts, invoices, resumes, prompts) into a judgement — a score, a classification, a prediction, or generated content.
tags: [ai]
timestamp: 2026-08-23T00:00:00Z
avoid: [AI simulation, AI heuristics, AI scoring]
related:
  - /adr/adr-0001-ai-inference-seam.md
---

# AI Inference

The capability that turns unstructured input into a judgement — a sentiment score, a contract risk rating, an OCR extraction, a resume parse, a stockout prediction, or generated SQL. It is a single domain concept regardless of which module requests it: CRM sentiment, accounting OCR, supply-chain risk, HR matching, and platform text-to-SQL are all the same capability exercised in different contexts.

_Avoid_: simulation, AI heuristic, AI helpers