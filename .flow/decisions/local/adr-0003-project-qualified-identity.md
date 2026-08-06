---
id: adr-0003-project-qualified-identity
type: decision
layer: local
status: accepted
title: Identity is project-qualified in the aggregated index
lineage: 2026-07-flow-genesis
reversibility: reversible
supersedes: null
superseded_by: null
decided: 2026-07-21
---

## Context

The desktop overlay aggregates many `.flow/` trees on one plane. Two projects can
hold the same local lineage or ADR id.

## Decision

The aggregated index key is `<project_id>:<local_id>`. Qualified ids are computed by
the core from each root's manifest and never stored in files.

## Consequences

Two projects cannot collide. Writes always resolve a qualified id back to exactly one
project root.
