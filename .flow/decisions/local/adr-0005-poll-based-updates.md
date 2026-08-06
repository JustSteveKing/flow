---
id: adr-0005-poll-based-updates
type: decision
layer: local
status: superseded
title: Overlay refreshes by polling the snapshot
lineage: 2026-07-flow-genesis
reversibility: reversible
supersedes: null
superseded_by: adr-0006-sse-live-updates
decided: 2026-07-21
---

## Context

The overlay needs to reflect external edits; the simplest mechanism is a timed
poll of the snapshot endpoint.

## Decision

The frontend polls `/api/snapshot` every 1.5s and re-renders on change.

## Consequences

Simple and robust, but edits lag by up to a poll and the request runs even when
nothing changed. Reversible; superseded once the watcher can push.
