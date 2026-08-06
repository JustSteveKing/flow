---
id: adr-0007-capacity-count-budget
type: decision
layer: local
status: accepted
title: Solo betting budget is a count-based capacity
lineage: 2026-07-flow-genesis
reversibility: reversible
supersedes: null
superseded_by: null
decided: 2026-07-21
---

## Context

Solo, there is no room of people to make the betting table a forcing function
(Decision 3).

## Decision

A cycle carries a `capacity` (count of concurrent bets, default 1). Placing a bet
that would breach capacity is refused; a pitch must be shelved to make room.

## Consequences

The scarcity of slots plus the circuit-breaker `ends` date does the forcing.
Reversible; capacity could later become appetite-weighted.
