---
id: adr-0006-sse-live-updates
type: decision
layer: local
status: accepted
title: Overlay refreshes via server-sent events
lineage: 2026-07-sse-live-updates
reversibility: reversible
supersedes: adr-0005-poll-based-updates
superseded_by: null
decided: 2026-07-21
---

## Context

Polling lags and wastes requests. The watcher already knows the exact moment the
index changes.

## Decision

The server pushes a change event over an SSE endpoint when the watcher re-indexes;
the frontend refreshes on that event, with a poll only as a reconnect fallback.

## Consequences

Near-instant refresh, no idle requests. Supersedes adr-0005. One-way server to
client only; no websockets (a lens does not need a back channel).
