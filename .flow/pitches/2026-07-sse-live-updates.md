---
lineage: 2026-07-sse-live-updates
type: pitch
status: bet
title: Server-sent live updates for the overlay
appetite: small-batch
problem: >
  The frontend polls the snapshot every 1.5s, so edits lag by up to a poll and the
  request runs even when nothing changed.
no_gos:
  - No websockets. One-way server-to-client is enough for a lens.
  - No change to the write path or the watcher's echo guard.
assumes_decisions:
  - adr-0006-sse-live-updates
rabbit_holes:
  - Reconnection storms if the window sleeps and wakes.
created: 2026-07-21
updated: 2026-07-21
---

# Server-sent live updates

Push a change event over an SSE endpoint when the watcher re-indexes, so the
window refreshes on edit rather than on a timer.
