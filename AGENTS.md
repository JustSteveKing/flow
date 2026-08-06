# AGENTS.md

Instructions for coding agents working in this repository. Humans should read
[README.md](README.md) first; this file assumes it.

## The one rule

**Agents propose. Humans commit state transitions.**

flow's whole premise is that a state change is a deliberate human act recorded
as a git commit (adr-0001). An agent that advances a document's `status` has
broken the product, not just the workflow. Draft the artefact, report what you
did, and stop at the gate.

Never run these, or their effects, on your own initiative:

| Forbidden | Why |
|-----------|-----|
| `flow table --add` / `--shelve` | Places or kills a bet. Human judgement. |
| `flow build --start` | Locks interfaces and no-gos. One-way. |
| `flow archive --cooldown` / `--close` | Moves or ends the cycle. |
| `flow decide --accept` | Makes a proposed ADR binding. |
| Hand-editing any `status:` field | Bypasses the gate and leaves no transition commit. |
| `git commit` unless explicitly asked | The commit *is* the transition record. |

Permitted, because they scaffold or report rather than commit the project to
anything: `flow shape`, `flow decide` raising a **proposed** ADR,
`flow build --hill`, and `flow status`.

One deliberate exception: **`flow spec --from` is a gate verb in the state
table, but it is the `flow-spec-proposer`'s explicit job.** It leaves the spec
at `specifying`, where interfaces and no-gos are still soft; the irreversible
step is `flow build --start`, which locks them. The two are separate verbs
precisely so that drafting a spec does not start the build. Run it only when
acting as the spec-proposer on an already-placed bet — never to move a pitch
that a human has not bet on.

If work seems to require crossing a no-go or changing a locked interface: stop
and report it as blocked. Do not do it and mention it afterwards.

## Commands

```sh
make check            # gofmt -l, go vet, go test -race — run this before you report done
make cli              # -> bin/flow
make test             # go test -race ./internal/... ./cmd/flow
make vet
make fmt              # gofmt -w
make desktop-frontend # npm install && vite build
make desktop          # frontend, then bin/flow-desktop
```

`make check` and friends deliberately scope to `./internal/... ./cmd/flow`.
They exclude `cmd/flow-desktop`, which `go:embed`s the untracked
`frontend/dist` and needs the Wails toolchain. Do not "fix" this by widening
them to `./...` — a fresh clone would stop building without Node.

## Architecture boundaries

```
cmd/flow/            CLI entrypoint
cmd/flow-desktop/    Wails v3 overlay + React frontend
internal/core/       Parsing, indexing, id resolution, write routing — the one model
internal/cli/        Cobra command surface over core
internal/desktop/    Workspace aggregation, HTTP API, project registry
internal/watch/      fsnotify watcher feeding live overlay updates
```

- **Domain logic lives in `internal/core` only** (adr-0002). The CLI and the
  overlay are thin clients. If you find yourself parsing frontmatter, resolving
  an id, or deciding a transition's legality anywhere else, it belongs in core.
- **Neither frontend carries its own copy of the model.** Duplicating a status
  enum or a hill rule into TypeScript is a no-go from the founding pitch.
- **Writes go through the store**, which writes atomically and preserves
  unknown frontmatter fields. Do not `os.WriteFile` a document directly.
- **The desktop app owns no truth** beyond its registry of project roots. Every
  read derives from the `.flow/` trees; every write resolves to exactly one
  project.

## Documents and the schema

[SCHEMA.md](SCHEMA.md) is the **frozen** frontmatter contract, schema version 1.
Changing it is an ADR, not an edit. The parser implements against it:

- A **MUST** violation makes the document invalid — dropped from the index with
  a recorded error.
- A **SHOULD** violation indexes with a warning.
- Unknown fields are preserved on write, ignored on read.

`.flow/` in this repo is the project's real working tree, not a fixture. flow
runs on flow. Treat those documents as production data: if a change to core
would alter how they parse, that is a breaking change.

Invalid-document fixtures belong in `testdata/invalid/`.

## Conventions

- **British English** in prose, comments, and doc text — *finalise*, *artefact*,
  *organise*, *behaviour*. Identifiers stay as they are.
- Standard Go test naming (`TestThingDoesX`), table tests via `t.Run`, helpers
  marked `t.Helper()`.
- New behaviour needs a test in the package that owns it.
- Comments explain *why*, and cite the governing ADR by id where one applies
  (e.g. `// adr-0002: core owns the model`). Match the surrounding density.
- Run `make fmt` before reporting done; CI fails on unformatted files.

## Specialised sub-agents

`.claude/agents/` defines five agents, one per state. Each produces an artefact
and stops at the next gate. Full descriptions in
[.claude/agents/README.md](.claude/agents/README.md).

| Agent | Trigger | Produces | Stops before |
|-------|---------|----------|--------------|
| `flow-shaping-critic` | pitch `shaping` | shaping review | betting |
| `flow-spec-proposer` | pitch `bet`, no spec | draft spec (`specifying`) | building |
| `flow-implementer` | spec `building`, one scope | code + that scope's hill | freeze, other scopes |
| `flow-spec-reviewer` | scope `done` | review, advisory only | merge and freeze |
| `flow-archiver` | cycle `cooldown` | freeze proposal | close |

If you are the implementer: build **one** scope, ignore the others, move its
hill with `flow build --spec <lineage> --hill <scope-id>=<0.0..1.0>`, and stop
at 1.0.

## Raising a decision

Hit an irreversible or precedent-setting fork? Raise a **proposed** ADR and
leave it proposed:

```sh
flow decide "<title>" --lineage <lineage> --reversibility one-way|reversible
```

Report it for a human to accept. Never accept your own.
