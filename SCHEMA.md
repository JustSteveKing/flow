---
lineage: 2026-07-flow-genesis
type: decision
id: adr-0004-frozen-contracts
layer: local
status: accepted
title: Frozen frontmatter contracts, schema version 1
reversibility: one-way
supersedes: null
superseded_by: null
decided: 2026-07-21
---

# flow contracts, schema version 1

This document freezes the frontmatter contract for every flow document type. It is the specification the Phase 1 parser implements against. A document that violates a rule marked **MUST** is invalid and is dropped from the index with a recorded error; a document that violates a **SHOULD** is indexed with a warning. Unknown fields are preserved on write and ignored on read.

All documents are UTF-8 markdown with a single leading YAML frontmatter block delimited by `---` lines. The body after the closing `---` is free markdown and is not parsed for meaning by the core.

## Shared field types

- **lineage id**: `^\d{4}-\d{2}-[a-z0-9]+(-[a-z0-9]+)*$`, for example `2026-07-passkey-login`. Minted once, immutable. Date prefix is the mint month.
- **project id**: `^[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$`, for example `juststeveking/flow`. Lives only in the manifest. Owner/name shape, two segments.
- **qualified id**: `<project_id>:<local_id>`, for example `juststeveking/flow:2026-07-passkey-login`. Never stored in files; computed by the core from the owning root's manifest. It is the aggregated index key.
- **adr id**: `^adr-\d{4}-[a-z0-9]+(-[a-z0-9]+)*$`, for example `adr-0012-webauthn-library`. Unique within its layer (baseline or local) inside one project.
- **cycle id**: `^\d{4}-C\d+$`, for example `2026-C3`.
- **date**: ISO 8601 date, `YYYY-MM-DD`.
- **hill**: float in the closed interval `[0.0, 1.0]`.

## Enumerations

- pitch `status`: `shaping | betting | shaped | bet | shelved`
- spec `status`: `specifying | building | frozen`
- scope `status`: `uphill | downhill | done`
- decision `status`: `proposed | accepted | superseded`
- decision `layer`: `baseline | local`
- decision `reversibility`: `one-way | reversible`
- cycle `status`: `active | cooldown | closed`
- appetite: `small-batch | big-batch`
- baseline `mode`: `spine | vendored`

## Document type: pitch

Path: `.flow/pitches/<lineage>.md`. One file per lineage.

| Field | Required | Type | Rule |
|-------|----------|------|------|
| `lineage` | MUST | lineage id | Matches the filename stem. Immutable. |
| `type` | MUST | const | Exactly `pitch`. |
| `status` | MUST | enum | pitch status. |
| `title` | MUST | string | Non-empty. |
| `appetite` | MUST | enum | appetite. |
| `problem` | SHOULD | string | One-line problem statement, not a solution. |
| `no_gos` | SHOULD | list of string | Explicit out-of-bounds. Carried into the spec. |
| `assumes_decisions` | SHOULD | list of adr id | Each MUST resolve to a decision in this project (local or baseline). |
| `rabbit_holes` | MAY | list of string | Known traps, kept visible for the shaping-critic. |
| `created` | MUST | date | |
| `updated` | MUST | date | Set on every write. |

Invariants: `status: bet` MUST imply the lineage appears in exactly one cycle's `bets`. `status: shelved` MUST imply it appears in some cycle's `shelved`.

## Document type: spec

Path: `.flow/specs/<lineage>.md`. One file per lineage, same lineage as its pitch.

| Field | Required | Type | Rule |
|-------|----------|------|------|
| `lineage` | MUST | lineage id | MUST match an existing pitch's lineage. |
| `type` | MUST | const | Exactly `spec`. |
| `status` | MUST | enum | spec status. |
| `cycle` | MUST | cycle id | MUST resolve to a cycle file. |
| `assumes_decisions` | SHOULD | list of adr id | Each MUST resolve. |
| `no_gos` | SHOULD | list of string | Inherited from the pitch, MAY be tightened, MUST NOT be loosened. |
| `interfaces` | MUST when status is building or frozen | list of interface | Locked at `flow build --start`. See below. |
| `scopes` | MUST | list of scope | At least one. See below. |
| `created` | MUST | date | |
| `updated` | MUST | date | |

`interface` object: `name` (MUST, string, unique within the spec), `contract` (MUST, string).

`scope` object:

| Field | Required | Type | Rule |
|-------|----------|------|------|
| `id` | MUST | string | Unique within the spec, kebab-case. |
| `title` | MUST | string | |
| `status` | MUST | scope status | |
| `hill` | MUST | hill | `0.0` when `uphill` and unstarted; `1.0` iff `status: done`. |
| `interfaces` | SHOULD | list of string | Names that MUST exist in the spec `interfaces` list. |

Invariants: while `status: specifying`, `interfaces` MAY be absent or in flux. On transition to `building`, `interfaces` and `no_gos` become locked and MUST NOT change (enforced by the write side, Phase 2). `hill: 1.0` iff scope `status: done`.

## Document type: decision (ADR)

Path: `.flow/decisions/baseline/<id>.md` or `.flow/decisions/local/<id>.md`.

| Field | Required | Type | Rule |
|-------|----------|------|------|
| `id` | MUST | adr id | Matches the filename stem. Unique within its layer. |
| `type` | MUST | const | Exactly `decision`. |
| `layer` | MUST | enum | MUST match the containing folder (`baseline` or `local`). |
| `status` | MUST | enum | decision status. |
| `title` | MUST | string | |
| `lineage` | SHOULD | lineage id | The work chain that emitted it. Baseline records MAY omit. |
| `reversibility` | MUST | enum | The primary bright-line axis (Decision 2). |
| `supersedes` | MUST | adr id or null | Backward pointer. |
| `superseded_by` | MUST | adr id or null | Forward pointer. |
| `decided` | MUST | date | |

Invariants: the chain MUST be consistent, if A `superseded_by` B then B `supersedes` A. A record with `status: superseded` MUST have a non-null `superseded_by`. A record with a non-null `superseded_by` MUST have `status: superseded`. A local record MAY supersede a baseline record (the escape hatch, Decision 4); a baseline record MUST NOT reference a local record. No cycles in the supersession chain.

## Document type: cycle

Path: `.flow/cycles/<id>.md`.

| Field | Required | Type | Rule |
|-------|----------|------|------|
| `id` | MUST | cycle id | Matches the filename stem. |
| `type` | MUST | const | Exactly `cycle`. |
| `status` | MUST | enum | cycle status. |
| `appetite_weeks` | MUST | int | Positive. The fixed appetite window. |
| `capacity` | SHOULD | int | Max concurrent bets (Decision 3, count-based budget). Positive. Absent means `1` (solo default). |
| `starts` | MUST | date | |
| `ends` | MUST | date | The circuit breaker date. MUST be after `starts`. |
| `bets` | SHOULD | list of bet | See below. |
| `shelved` | MAY | list of shelved | See below. |

`bet` object: `lineage` (MUST, lineage id, resolves to a pitch with `status: bet`), `appetite` (MUST, appetite), `placed` (MUST, date).

`shelved` object: `lineage` (MUST, lineage id).

Invariants: a lineage MUST NOT appear in both `bets` and `shelved` of the same cycle. The count of `bets` MUST NOT exceed `capacity` (the count-based appetite budget, Decision 3). The write side refuses a bet that would breach capacity; a person must shelve to make room.

## Document type: manifest

Path: `.flow/manifest.yaml`. Pure YAML, no markdown body. Read first by the core.

| Field | Required | Type | Rule |
|-------|----------|------|------|
| `version` | MUST | int | Schema version. `1` for this contract. |
| `project` | MUST | string | Human name. |
| `project_id` | MUST | project id | Globally unique across a user's registry. |
| `baseline.mode` | MUST | enum | `spine` or `vendored`. |
| `baseline.ref` | MUST when mode is spine | string | Submodule path or remote. |
| `id_prefix` | SHOULD | string | Local ADR id prefix. Default `adr`. |

## Registry (desktop, not a `.flow/` document)

Path: `~/.config/flow/registry.yaml`. The one piece of app-owned state.

| Field | Required | Type | Rule |
|-------|----------|------|------|
| `version` | MUST | int | `1`. |
| `roots` | MUST | list of root | Each `path` (MUST, absolute) and `project_id` (MUST, project id). |

Invariant: `project_id` values MUST be unique across roots (identity cannot collide).

## Cross-document referential rules (index-level)

1. Every spec `lineage` resolves to a pitch of the same lineage.
2. Every `assumes_decisions` id resolves to a decision in the same project (local shadows baseline on id clash).
3. Every spec `cycle` resolves to a cycle file.
4. Every cycle `bet.lineage` resolves to a pitch; that pitch's `status` is `bet`.
5. The supersession graph across baseline and local is acyclic; each chain has exactly one head (non-superseded) record.
