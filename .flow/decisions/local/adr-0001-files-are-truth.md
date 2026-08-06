---
id: adr-0001-files-are-truth
type: decision
layer: local
status: accepted
title: Files in git are the single source of truth
lineage: 2026-07-flow-genesis
reversibility: one-way
supersedes: null
superseded_by: null
decided: 2026-07-21
---

## Context

flow could keep an index or cache as the authoritative store for speed.

## Decision

Files in the `.flow/` tree, committed to git, are the only source of truth. Every
read is derivable from files; every write lands in a file and is committed. Any
in-memory index is a rebuildable projection, never authoritative.

## Consequences

The desktop app is a lens with no truth of its own. Git history is the transition
event log. Precedent-setting for every later component: nothing caches ahead of disk.
