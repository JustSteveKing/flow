---
lineage: 2026-07-flow-genesis
type: pitch
status: bet
title: flow, an SDLC operating system
appetite: big-batch
problem: >
  Development methodology lives in people's heads and scattered docs, not in
  version control, so it cannot be queried, enforced, or traced.
no_gos:
  - No database. Files in git are the only source of truth.
  - No app-owned state beyond the desktop registry of project roots.
assumes_decisions:
  - adr-0001-files-are-truth
  - adr-0002-single-shared-core
  - adr-0003-project-qualified-identity
rabbit_holes:
  - Cross-repo baseline write-routing.
  - fsnotify event storms during git operations.
created: 2026-07-21
updated: 2026-07-21
---

# flow, an SDLC operating system

See `PLAN.md` at the repository root for the full shaped pitch. This is the
dogfooded lineage head for flow building itself.
