---
lineage: 2026-07-flow-genesis
type: pitch
status: shaping
title: flow, an SDLC operating system
appetite: big-batch
no_gos:
  - No database. Files in git are the only source of truth.
  - No app-owned state beyond the desktop registry of project roots.
  - No agent walks through a gate. Agents propose and stop.
  - Two frontends never carry their own copy of the domain model.
assumes_decisions:
  - adr-0001-files-are-truth
  - adr-0002-single-shared-core
  - adr-0003-project-qualified-identity
created: 2026-07-21
updated: 2026-07-21
---

# flow

## 1. Overview

flow is a file-based, git-tracked operating system for running software development. Every unit of work is a markdown document with YAML frontmatter, living in a `.flow/` tree inside the project repository. Work moves through a state machine whose transitions are human git commits. There is no database and no server-owned truth. Git history is the event log. A Go CLI and a desktop viewer both sit on top of one shared Go core, so the model cannot drift between them.

flow fuses three methodologies into one lineage of documents:

- **Shape Up** is the container. It decides whether and roughly how work happens: a fixed appetite, a betting step, hill charts, and a circuit breaker. When the appetite expires the work is re-shaped, not extended.
- **Spec-driven development** is the buildable surface inside a cycle. It decides exactly what gets built.
- **Architecture Decision Records** are a cross-cutting event, not a phase. They record why something was locked in and they constrain future specs. Their lifecycle is proposed, then accepted, then superseded.

### Guiding principles

1. **Files are truth.** If you delete the desktop app you lose pixels and nothing else. Every read is derivable from the `.flow/` trees. Every write lands in a file and is committed to a repository.
2. **Git is the event log.** Frontmatter `status` is the current state. The sequence of commits that changed it is the transition history. flow adds no second ledger.
3. **One lineage per idea.** A pitch, the spec it becomes, and the decisions it emits all carry the same lineage id, so the chain is traceable from problem to shipped code.
4. **Transitions are human gates.** Sub-agents draft artefacts and stop. A person commits the state change.
5. **One core, two lenses.** Indexing, parsing, id resolution, and write-routing live in a single Go package. The CLI and the desktop app are thin clients over it.

## 2. Methodology and state machine

### States

Work flows through six states. Each is a `status` value in frontmatter and each transition between them is a commit.

```
shaping -> betting -> specifying -> building -> cooldown -> archived
```

- **shaping**: a pitch exists. Problem, appetite, solution sketch, no-gos, and rabbit holes are being argued. Output is a shaped pitch.
- **betting**: shaped pitches compete for a slot in the next cycle. Output is a bet (a pitch bound to a cycle) or a shelved pitch.
- **specifying**: the bet becomes a spec. Scopes are cut, interfaces and no-gos are fixed, scope bodies are left just-in-time (see Open decision 1).
- **building**: scopes are worked. Each carries a hill position that moves from uphill (figuring out) to downhill (executing) to done.
- **cooldown**: the cycle winds down. Specs are frozen, ADRs are finalised, loose ends are recorded rather than chased.
- **archived**: the lineage is closed. Documents stay in the tree, read-only by convention, fully queryable.

### Transitions as gates

Each arrow is a deliberate commit by a person, usually invoked through a CLI verb. The verb mutates frontmatter and stages the file. The commit is the transition record. flow never auto-advances a state.

| From | To | Gate verb | What changes |
|------|-----|-----------|--------------|
| (none) | shaping | `flow shape` | Creates a pitch with a fresh lineage id, status `shaping`. |
| shaping | betting | `flow table --add` | Pitch marked shaped and entered on the cycle table. |
| betting | specifying | `flow spec --from <lineage>` | Bet bound to cycle. Spec scaffold created, status `specifying`. |
| specifying | building | `flow build --start` | Interfaces and no-gos locked, scopes opened, status `building`. |
| building | cooldown | `flow archive --cooldown` | Cycle enters cooldown, hill frozen. |
| cooldown | archived | `flow archive --close` | Specs frozen, proposed ADRs finalised, lineage closed. |

A pitch can also move to `shelved` from betting, and the whole cycle honours the circuit breaker: when a cycle's appetite window ends and scopes are still uphill, flow does not extend. `flow archive --cooldown` closes the cycle and the unfinished lineage returns to `shaping` for a fresh pitch.

### ADR lifecycle, running across all phases

ADRs are emitted at any point, most often during shaping (a bet assumes a decision) and specifying (a scope forces one), sometimes during building (an implementer hits an irreversible fork). They are not a state in the work machine. They have their own three-state lifecycle:

```
proposed -> accepted -> superseded
```

- **proposed**: written but not binding. A spec may reference it, but reviewers know it can still move.
- **accepted**: locked. Future specs must respect it or supersede it.
- **superseded**: replaced by a newer ADR. The `supersedes` and `superseded_by` fields form a chain, so no history is lost and the current position is always the head of each chain.

An ADR carries its own lineage link back to the work that produced it, so "why was this locked in" resolves to a pitch and a spec, and "what does this pitch assume" resolves to a set of ADRs.

## 3. Substrate and contracts

### The `.flow/` tree

```
.flow/
  manifest.yaml           # project identity and configuration
  cycles/
    2026-C3.md            # one file per cycle, holds the betting table and appetite
  pitches/
    <lineage>.md          # one pitch per lineage
  specs/
    <lineage>.md          # one spec per lineage, cut into scopes
  decisions/
    baseline/             # durable, cross-project ADRs (see Open decision 4)
      adr-0001.md
    local/                # project-specific ADRs, may supersede baseline
      adr-0044.md
  reviews/
    <lineage>-<agent>.md  # sub-agent output, advisory, never a gate
```

Pitches, specs, and their decisions are joined by a shared `lineage` id, not by directory nesting, so the same idea is one chain across four folders. The lineage id is minted once at `flow shape` and is immutable.

### Lineage id format

`YYYY-MM-<slug>` where slug is a short stable token derived from the title at creation, for example `2026-07-passkey-login`. Human-readable, sortable, collision-checked against the index at mint time. This is the local id. The globally unique id used by the desktop overlay is `project_id:lineage` (see section 6).

### Frontmatter contracts

Every document is markdown with a YAML frontmatter block. Fields below marked required must be present for the core to index the file; unknown fields are preserved on write but ignored.

#### Pitch

```yaml
lineage: 2026-07-passkey-login      # required, immutable
type: pitch                          # required
status: shaping                      # shaping | betting | shaped | bet | shelved
title: Passkey login
appetite: small-batch                # small-batch | big-batch (Shape Up appetites)
problem: >
  One-line statement of the raw problem, not the solution.
no_gos:                              # explicit out-of-bounds, carried into the spec
  - No account recovery flow this cycle.
  - No migration of existing password users.
assumes_decisions:                   # ADR ids this pitch depends on
  - adr-0012-webauthn-library
rabbit_holes:                        # known traps, kept visible for the critic
  - Cross-device sync of credentials.
created: 2026-07-21
updated: 2026-07-21
```

#### Spec

```yaml
lineage: 2026-07-passkey-login       # required, same as the pitch
type: spec                           # required
status: specifying                   # specifying | building | frozen
cycle: 2026-C3                       # the cycle this bet lives in
assumes_decisions:
  - adr-0012-webauthn-library
no_gos:                              # inherited from the pitch, may be tightened
  - No account recovery flow this cycle.
interfaces:                          # fixed at flow build --start (Open decision 1)
  - name: RegisterCredential
    contract: >
      POST /auth/passkey/register, body {userId, attestation}, returns {credentialId}.
scopes:
  - id: scope-register
    title: Credential registration
    status: uphill                   # uphill | downhill | done
    hill: 0.15                        # 0.0 unstarted, 0.5 crest, 1.0 done
    interfaces: [RegisterCredential]  # which fixed interfaces this scope owns
  - id: scope-authenticate
    title: Assertion and login
    status: uphill
    hill: 0.0
created: 2026-07-21
updated: 2026-07-21
```

#### Decision (ADR)

```yaml
id: adr-0012-webauthn-library        # required, unique within its layer
type: decision                       # required
layer: local                         # baseline | local
status: accepted                     # proposed | accepted | superseded
title: Use the go-webauthn library
lineage: 2026-07-passkey-login       # the work chain that emitted it, if any
reversibility: one-way               # one-way | reversible (the bright-line test)
supersedes: null                     # adr id or null
superseded_by: null                  # adr id or null, the forward pointer
decided: 2026-07-21
context: >
  Why the decision was forced.
decision: >
  What was chosen.
consequences: >
  What this constrains going forward.
```

#### Cycle

```yaml
id: 2026-C3                          # required
type: cycle                          # required
status: active                       # active | cooldown | closed
appetite_weeks: 6                    # the fixed appetite window
starts: 2026-07-21
ends: 2026-09-01                     # the circuit breaker date
bets:                                # the betting table for this cycle
  - lineage: 2026-07-passkey-login
    appetite: big-batch
    placed: 2026-07-21
shelved:
  - lineage: 2026-06-audit-log
```

#### Manifest

```yaml
version: 1                           # schema version, for migration
project: flow                        # human name
project_id: juststeveking/flow       # project-qualified identity, globally unique
baseline:
  mode: spine                        # spine | vendored (Open decision 4)
  ref: ../flow-baseline              # path or remote, meaning depends on mode
id_prefix: adr                       # local ADR id prefix
```

The manifest is the one file the core reads first. `project_id` is the root of project-qualified identity and must be unique across a user's registry.

## 4. Sub-agents and gates

Every sub-agent obeys the same contract: it is triggered by a state, reads a fixed set of inputs, produces exactly one artefact, and stops at a named gate without committing the transition. Its output lands in `reviews/` or as a draft file, never as a state change. A person reads it and runs the gate verb.

### shaping-critic

- **Trigger**: a pitch in `shaping`.
- **Inputs**: `pitches/<lineage>.md`, the assumed ADRs, the current cycle appetite.
- **Produces**: `reviews/<lineage>-shaping-critic.md`, a critique that pushes on appetite realism, unlisted rabbit holes, and no-gos that are missing or too loose.
- **Gate**: stops before betting. It cannot mark the pitch shaped. It recommends shape or re-shape.

### spec-proposer

- **Trigger**: a pitch with status `bet` that has no spec yet.
- **Inputs**: the pitch, its no-gos, its assumed ADRs, the cycle.
- **Produces**: a draft `specs/<lineage>.md` with proposed scopes, proposed interfaces, and inherited no-gos, status left at `specifying`.
- **Gate**: stops before building. It never locks interfaces and never opens scopes for work.

### implementer

- **Trigger**: a spec in `building` with a scope assigned to it.
- **Inputs**: exactly one scope, its owned interfaces, the no-gos, the assumed ADRs.
- **Produces**: code changes for that one scope, plus an updated `hill` and `status` on that scope in the spec frontmatter.
- **Gate**: stops at scope done. It may raise a proposed ADR if it hits an irreversible fork, but it never freezes the spec, never touches other scopes, and never moves the cycle.

### spec-reviewer

- **Trigger**: a scope reaching `hill: 1.0` or status `done`.
- **Inputs**: the spec, the scope, the diff for that scope, the assumed ADRs.
- **Produces**: `reviews/<lineage>-spec-reviewer.md`, checking the build against the fixed interfaces and no-gos and flagging scope creep.
- **Gate**: stops before merge and before freeze. Advisory only.

### archiver

- **Trigger**: a cycle entering `cooldown`.
- **Inputs**: all specs in the cycle, all ADRs referenced by them, the cycle file.
- **Produces**: a freeze proposal that would set specs to `frozen` and move `proposed` ADRs to `accepted` or `superseded`, written as a draft for review.
- **Gate**: stops at the archive commit boundary. It proposes the freeze. A person runs `flow archive --close` to commit it.

## 5. CLI

A Go CLI built on Cobra. All verbs are thin wrappers over the shared core; each resolves a lineage or cycle, mutates frontmatter, stages the file, and leaves the commit to the user (or makes it with an explicit `--commit`). The core, not the CLI, owns parsing and write-routing.

- `flow init` writes `manifest.yaml`, creates the `.flow/` tree, and mints `project_id`.
- `flow shape <title>` mints a lineage and creates a pitch in `shaping`.
- `flow table` shows the current cycle's betting table. `flow table --add <lineage>` marks a pitch shaped and places the bet. `flow table --shelve <lineage>` shelves it.
- `flow spec --from <lineage>` binds the bet to the cycle and scaffolds a spec in `specifying`.
- `flow decide <title>` creates a `proposed` ADR in `local/`, links its lineage, and sets `reversibility`. `flow decide --accept <id>` and `flow decide --supersede <old> --by <new>` walk the ADR lifecycle and maintain both ends of the chain.
- `flow build --start <lineage>` locks interfaces and no-gos and moves the spec to `building`. `flow build --hill <scope> <value>` updates a scope's hill position.
- `flow archive --cooldown <cycle>` enters cooldown. `flow archive --close <cycle>` freezes specs, finalises ADRs, and closes lineages.
- `flow status` and `flow ls` are read-only lenses over the index, mirroring what the desktop shows.

Every mutating verb prints the exact file diff it will write before writing, honouring the propose-then-commit ethos even at the CLI.

## 6. Desktop overlay

The desktop viewer is a workspace overlay across many `.flow/` trees on one plane. It is a lens. It holds exactly one piece of legitimate state.

### The registry

A single inspectable file, `~/.config/flow/registry.yaml`, lists the project roots to watch:

```yaml
version: 1
roots:
  - path: /Users/steve/Developer/GitHub/juststeveking/flow
    project_id: juststeveking/flow
  - path: /Users/steve/work/acme/api
    project_id: acme/api
```

This is the only state the app owns. It is user-editable, diffable, and losing it costs nothing but the list of what to watch. It is deliberately not inside any `.flow/` tree because it spans them.

### Aggregation and write-routing

- **Reads aggregate.** The core indexes every root in the registry into one in-memory index, keyed by `project_id:lineage`. Cross-project views query this index.
- **Writes resolve to one project.** Every mutation carries a project-qualified id. The core maps `project_id` back to its root path, writes the file there, and commits in that project's own repository. A write never spans two repositories and two projects can never collide because identity is always qualified.

### Project-qualified identity

The globally unique key is `project_id:lineage`, for example `acme/api:2026-07-passkey-login`. ADR ids are qualified the same way, `acme/api:adr-0012`. Two projects may both hold `adr-0012` locally without clashing in the aggregated index.

### The three cross-project views

1. **Unified betting table.** Every active cycle's bets and shelved pitches on one plane, grouped by project, so appetite commitments across the portfolio are visible at once.
2. **Portfolio hill chart.** Every building scope from every project plotted on one hill, so what is still uphill (unfigured-out risk) across all projects is visible in a glance.
3. **ADR supersession graph.** The `supersedes` and `superseded_by` chains across baseline and local layers, rendered as a graph, so the live head of each decision and the trail behind it are both legible.

## 7. Filesystem and sync

The viewer must reflect edits made in an IDE, so it watches the filesystem with fsnotify and re-indexes on change. The watching layer is designed around the known hazards.

- **Atomic saves.** Editors write to a temp file and rename over the target, so a naive per-file watch misses the real write. Watch directories, not files, and treat `CREATE`, `WRITE`, and `RENAME` as one logical "this file changed" signal.
- **Non-recursive fsnotify.** fsnotify does not recurse. On startup, walk each `.flow/` tree and add a watch per subdirectory. On any directory `CREATE`, add a watch to the new subdirectory before indexing its contents.
- **Git event storms.** A checkout, rebase, or pull rewrites many files at once. Debounce events per root with a short quiet-period timer and coalesce them into a single re-index pass, rather than re-indexing per event.
- **Half-written files.** A file caught mid-write yields truncated YAML. Parse defensively: on a frontmatter parse error, keep the last good index entry and retry after the debounce window rather than dropping the document.
- **Ignore noise.** Skip `.git` internals, editor scratch files (`*.swp`, `*~`, `.DS_Store`, `*.tmp`), and anything outside a `.flow/` subtree.
- **inotify limits.** On Linux the inotify watch count is bounded. Watch only `.flow/` subtrees, never whole repositories, so the watch budget scales with flow documents and not with source trees.
- **No echo.** App-originated writes must not bounce back as external changes. Before writing, record the target path and an expected signature in an in-flight set; when the resulting fsnotify event arrives, match and swallow it. Only unmatched events are treated as external and re-indexed.

The watching layer lives in the shared core so the CLI can reuse the same indexer without the watcher, and the desktop adds only the fsnotify loop on top.

## 8. Open decisions

Each is marked OPEN and awaits sign-off. Options and a recommendation are given; none is baked into the rest of the plan beyond the assumption noted.

### Decision 1: how much of a spec is fixed before building begins

**RESOLVED 2026-07-21: Option C.** Interfaces and no-gos fixed at `flow build --start`, scope bodies just-in-time.

- **Option A, fix everything.** Full spec, every scope body written, before a line is built. Kills Shape Up's variable scope; the appetite becomes meaningless because there is nothing left to trade.
- **Option B, fix nothing.** Only a title and appetite. Agents and builders have no contract to build against and no way to detect scope creep.
- **Option C, fix interfaces and no-gos, leave scope bodies just-in-time.** The boundary of the work (the interfaces scopes must honour, and the no-gos they must not cross) is locked at `flow build --start`. How each scope is built internally is decided as the scope is worked, and the hill chart tracks that discovery.

**Recommendation: Option C.** It preserves the appetite as a real constraint (scope bodies flex, the boundary does not), gives implementer agents a hard contract (the fixed interfaces they own and the no-gos they cannot cross), and makes scope creep detectable by the spec-reviewer (anything crossing a fixed interface or a no-go is creep). The rest of this plan assumes C, and the spec frontmatter reflects it: `interfaces` and `no_gos` are locked, `scopes[].hill` tracks the just-in-time bodies.

### Decision 2: the bright line between an ADR and spec text

**RESOLVED 2026-07-21: reversibility primary, blast radius second trigger.** ADR if hard to reverse or precedent-setting.

- **Proposed test: reversibility.** If undoing the choice later is cheap and local, it is spec text. If undoing it is expensive, cross-cutting, or one-way (a data format, a public contract, a dependency the codebase organises itself around), it is an ADR.
- **Pressure test.** Reversibility is a spectrum, not a switch. Some choices are reversible in principle but precedent-setting in practice (everyone copies the first auth pattern). Some are cheap to reverse in code but expensive in migration (a persisted enum). A pure reversibility test under-captures precedent and over-trusts "we can change it later".
- **Refinement.** Use reversibility as the primary test and add a second trigger: **blast radius**. A choice becomes an ADR if it is hard to reverse *or* if it sets a precedent other work will follow. The `reversibility` frontmatter field records the primary axis; the ADR body's `consequences` section captures precedent.

**Recommendation: reversibility as the primary bright line, with blast radius as a second trigger.** This keeps the common case crisp (one-way choices are always ADRs) while catching the precedent case the pure test misses.

### Decision 3: how the solo betting table earns its keep

**RESOLVED 2026-07-21: Option C** (ledger plus hard appetite budget), encoded as a count-based `capacity` on the cycle. `flow table --add` refuses a bet that would breach capacity; a pitch must be shelved to make room. Unset capacity means the solo default of one concurrent bet. Shape Up's betting table is a forcing function because a room of people must agree to spend the appetite. Solo, there is no room.

- **Option A, drop it.** A solo builder just picks the next thing. Loses the circuit breaker and the explicit kill decision.
- **Option B, keep it as a ritual ledger.** The table is a written commitment: the builder must record, per cycle, exactly which bets are placed and which pitches are shelved, and cannot start building without a placed bet.
- **Option C, ledger plus a hard appetite budget.** As B, plus the cycle carries a fixed capacity, and placing a bet that exceeds remaining capacity is refused by `flow table --add`. The forcing function becomes the scarcity of the budget, not the scarcity of people.

**Recommendation: Option C.** The value of the betting table is that it forces an explicit "not this, that" decision under a real constraint. For a solo builder the constraint is time, so encode the appetite budget as a hard cap the CLI enforces, and make shelving a first-class recorded act (`flow table --shelve`) rather than a silent non-choice. The circuit breaker (the `ends` date) then does the rest of the forcing.

### Decision 4: where baseline ADRs physically live

**RESOLVED 2026-07-21: Option A.** Spine as read-only git submodule, local supersession as the escape hatch. This changes what "resolve a write back to one project" means for baseline records.

- **Option A, a shared spine repo.** Baseline lives in one repository (a git submodule or a sibling checkout referenced by `manifest.baseline.ref`). Every project reads it. Writes to baseline resolve to the spine repo, not the project, which breaks the clean "one write, one project repo" rule and needs a defined commit target.
- **Option B, vendored into each project.** Each project holds its own copy of `decisions/baseline/`. Writes always resolve to the project, preserving the rule, but baseline records drift between projects and there is no single source of a shared decision.
- **Trade-off.** A carries authority (one true baseline) at the cost of a second write target and a cross-repo coupling the live-sync layer must also watch. B carries autonomy and a clean write model at the cost of drift and a sync problem (how does a corrected baseline reach every project).

**Recommendation: Option A, spine as a read-only git submodule, with local supersession as the escape hatch.** Projects read baseline but never write it directly; a project that needs to diverge writes a `local/` ADR that supersedes the baseline record, which keeps every project write resolving to the project's own repo. Baseline edits happen only in the spine repo, through the same `flow decide` verb run with the spine as the working project. This preserves the write rule for the common case and confines the cross-repo write to an explicit, rare act. Flagged rather than assumed because it commits the sync layer to watching the submodule and defines "baseline write target" as the spine repo.

### Decision 5: desktop shell and core coupling

**RESOLVED 2026-07-21: Option A.** Wails v3 with the core in-process; CLI links the same core as a library.

- **Option A, Wails v3 with the core in-process.** The desktop shell embeds the Go core directly. Simplest to build, one binary, no IPC. The CLI links the same core as a library. Concurrent writers (CLI and app open at once) coordinate only through the filesystem and git, so the write-echo guard and git-level discipline carry the whole load.
- **Option B, a headless Go daemon.** A long-running daemon owns the core, the index, and the watcher. The CLI and any shell are clients over a local socket. One process owns writes, so concurrent-writer coordination is centralised and the live-sync index is shared, not rebuilt per client. Cost is a daemon lifecycle, an IPC contract, and a harder "files are truth" story if the daemon ever caches ahead of disk.

**Recommendation: Option A, Wails v3 in-process, for the first shippable system.** It keeps the binary count low, keeps the "files are truth" story airtight (no process caches ahead of disk; every read re-derives from files), and the concurrent-writer problem is already bounded by the write-echo guard plus git being the arbiter of conflicts. Revisit Option B only if a real multi-client need appears (a web shell, or several editors on one index). Flagged because it commits both frontends to linking the core as a library and pushes all concurrent-writer safety onto the filesystem and git rather than a single owning process.

## 9. Build phases

Dependency-ordered, each phase independently shippable, built from the core outward.

**Phase 0, contracts frozen.** Lock the frontmatter schemas for all five document types and the manifest, plus the lineage and project-qualified id formats. Output is a schema document and fixtures. Nothing else can start until these are stable. Blocked on sign-off of Decisions 1 and 2 (they shape the spec and ADR contracts).

**Phase 1, the core, read side.** The domain package: frontmatter parsing, the `.flow/` indexer, lineage and id resolution, and the aggregated index keyed by `project_id:lineage`. Ships as a library with a query API and a test suite over the Phase 0 fixtures. No writes yet.

**Phase 2, the core, write side.** Frontmatter mutation, the state-machine transition rules, and write-routing (resolve a qualified id back to one project root). Ships the propose-then-write path every gate verb will reuse. Depends on Decision 4 (baseline write target) and Decision 5 (in-process vs daemon shapes the write API).

**Phase 3, the CLI.** Cobra verbs over the Phase 2 core: `init`, `shape`, `table`, `spec`, `decide`, `build`, `archive`, `status`. This is the first end-to-end usable flow: shape, bet, spec, build, archive, all from the terminal, all committing to git. Independently shippable and dogfoodable on its own.

**Phase 4, the watching layer.** fsnotify over `.flow/` subtrees with every hazard from section 7 handled: directory watches, recursive walk-and-add, debounce and coalesce, defensive parsing, ignore rules, inotify budget, write-echo guard. Ships as an addition to the core that the CLI can ignore and the desktop will depend on.

**Phase 5, the desktop overlay.** Wails v3 shell (pending Decision 5) over the core plus watcher. The registry file, cross-project aggregation, project-qualified write-routing, and the three views: unified betting table, portfolio hill chart, ADR supersession graph. A lens with no truth of its own.

**Phase 6, the sub-agents.** The five agents from section 4, each triggered by a state, each producing one artefact into `reviews/` or as a draft, each stopping at its gate. Built last because each depends on the contracts, the core, and the verbs already existing to define its inputs and its stop point.
