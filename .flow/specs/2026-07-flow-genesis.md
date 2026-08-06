---
lineage: 2026-07-flow-genesis
type: spec
status: building
cycle: 2026-C1
assumes_decisions:
  - adr-0001-files-are-truth
  - adr-0002-single-shared-core
  - adr-0003-project-qualified-identity
  - adr-0004-frozen-contracts
  - adr-0007-capacity-count-budget
no_gos:
  - No database. Files in git are the only source of truth.
  - No app-owned state beyond the desktop registry.
interfaces:
  - name: ProjectIndex
    contract: LoadProject(root) reads a .flow tree into a rebuildable index.
  - name: Store
    contract: Plan/Apply route a document to one project and write it atomically.
  - name: Watcher
    contract: fsnotify over .flow subtrees, debounced, with an app-write echo guard.
scopes:
  - id: phase-0-contracts
    title: Freeze frontmatter contracts
    status: done
    hill: 1.0
  - id: phase-1-core-read
    title: Core read side
    status: done
    hill: 1.0
    interfaces: [ProjectIndex]
  - id: phase-2-core-write
    title: Core write side
    status: done
    hill: 1.0
    interfaces: [Store]
  - id: phase-3-cli
    title: Cobra CLI lifecycle
    status: done
    hill: 1.0
  - id: phase-4-watcher
    title: fsnotify watching layer
    status: done
    hill: 1.0
    interfaces: [Watcher]
  - id: phase-5-desktop
    title: Wails v3 overlay and views
    status: downhill
    hill: 0.8
  - id: phase-6-subagents
    title: Gate-stopping sub-agents
    status: done
    hill: 1.0
created: 2026-07-21
updated: 2026-07-21
---

# Spec: flow

The seven build phases as scopes. Interfaces above are the fixed boundary
(Decision 1); scope bodies were figured out just in time.
