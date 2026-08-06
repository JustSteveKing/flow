---
id: adr-0004-frozen-contracts
type: decision
layer: local
status: accepted
title: Frozen frontmatter contracts, schema version 1
lineage: 2026-07-flow-genesis
reversibility: one-way
supersedes: null
superseded_by: null
decided: 2026-07-21
---

## Context

Two frontends and a set of agents build against the document shapes; they need a
stable target.

## Decision

Freeze the frontmatter contract for all five document types at schema version 1,
recorded in `SCHEMA.md`. Unknown fields are preserved and ignored.

## Consequences

One-way: changing a contract is a migration, not an edit. Precedent for every
component that parses or writes flow documents.
