---
id: adr-0099-broken-chain
type: decision
layer: local
status: accepted
title: superseded_by set but status not superseded
lineage: 2026-07-flow-genesis
reversibility: reversible
supersedes: null
superseded_by: adr-0100-replacement
decided: 2026-07-21
---

Non-null `superseded_by` requires `status: superseded`; invariant violation.
