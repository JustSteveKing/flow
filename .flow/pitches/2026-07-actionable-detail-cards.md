---
lineage: 2026-07-actionable-detail-cards
type: pitch
status: bet
title: Actionable detail cards in the overlay
appetite: small-batch
problem: >
  The detail panel shows per-project cycles, specs, and ADRs but is read only, so
  every state change still means dropping to the CLI.
no_gos:
  - No new truth in the app. Every action routes to files and commits in the project repo.
  - No bulk or cross-project actions from a single card.
assumes_decisions:
  - adr-0001-files-are-truth
  - adr-0002-single-shared-core
rabbit_holes:
  - Optimistic UI that drifts from the on-disk state before the reindex lands.
created: 2026-07-21
updated: 2026-07-21
---

# Actionable detail cards

Let a detail card accept or supersede an ADR and start building a spec, each
routed through the existing write path so the files stay the source of truth.
