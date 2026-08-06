---
lineage: 2026-07-baseline-spine
type: pitch
status: shelved
title: Baseline spine repo as a submodule
appetite: big-batch
problem: >
  Baseline ADRs are vendored per project (Decision 4 default), so a corrected
  baseline record does not reach other projects.
no_gos:
  - No writes to the spine from a project; divergence is a local supersession only.
rabbit_holes:
  - Submodule ref drift across projects on different baseline commits.
created: 2026-07-21
updated: 2026-07-21
---

# Baseline spine repo

Move baseline ADRs into a read-only submodule (Decision 4, Option A). Shelved this
cycle: one-way, better made once the local layer has settled.
