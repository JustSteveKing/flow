---
lineage: 2026-07-actionable-detail-cards
type: spec
status: building
cycle: 2026-C1
assumes_decisions:
  - adr-0001-files-are-truth
  - adr-0002-single-shared-core
no_gos:
  - No new truth in the app. Every action routes to files and commits in the project repo.
  - No bulk or cross-project actions from a single card.
interfaces:
  - name: AcceptADR
    contract: POST /api/adr/accept {projectId, id} advances a proposed ADR to accepted.
  - name: SupersedeADR
    contract: POST /api/adr/supersede {projectId, older, newer} links the chain and freezes older.
  - name: StartBuild
    contract: POST /api/spec/build {projectId, lineage} locks interfaces and moves a spec to building.
scopes:
  - id: scope-accept
    title: Accept an ADR from its card
    status: downhill
    hill: 0.6
    interfaces: [AcceptADR]
  - id: scope-supersede
    title: Supersede an ADR from its card
    status: uphill
    hill: 0.3
    interfaces: [SupersedeADR]
  - id: scope-start-build
    title: Start building a spec from its card
    status: uphill
    hill: 0.1
    interfaces: [StartBuild]
created: 2026-07-21
updated: 2026-07-21
---

# Spec: actionable detail cards

Three write actions surfaced on detail cards, each a thin route over the existing
core write path.
