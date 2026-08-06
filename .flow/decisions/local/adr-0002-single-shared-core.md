---
id: adr-0002-single-shared-core
type: decision
layer: local
status: accepted
title: One shared Go core drives both frontends
lineage: 2026-07-flow-genesis
reversibility: one-way
supersedes: null
superseded_by: null
decided: 2026-07-21
---

## Context

A CLI and a desktop viewer could each own their own model of the substrate.

## Decision

Indexing, frontmatter parsing, id resolution, and write-routing live in one Go
domain package. The CLI and the Wails desktop shell are thin clients that link it
as a library (Decision 5, in-process).

## Consequences

The model cannot drift between frontends. Precedent-setting: no frontend reimplements
domain logic.
