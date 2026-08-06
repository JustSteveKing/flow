---
lineage: 2026-07-flow-genesis
type: spec
status: building
cycle: 2026-C1
interfaces:
  - name: Index
    contract: BuildIndex(roots) returns Index
scopes:
  - id: scope-index
    title: Indexer
    status: uphill
    hill: 1.7
created: 2026-07-21
updated: 2026-07-21
---

`hill` outside [0.0, 1.0]; MUST be dropped with a recorded error.
