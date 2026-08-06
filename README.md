# flow

A file-based, git-tracked operating system for running software development.

Every unit of work is a markdown document with YAML frontmatter, living in a
`.flow/` tree inside your project repository. Work moves through a state machine
whose transitions are human git commits. There is no database and no
server-owned truth — git history *is* the event log. A Go CLI and a desktop
overlay both sit on top of one shared Go core, so the model cannot drift between
them.

flow fuses three methodologies into one lineage of documents:

- **Shape Up** is the container — fixed appetite, a betting step, hill charts, a
  circuit breaker. When the appetite expires, work is re-shaped, not extended.
- **Spec-driven development** is the buildable surface inside a cycle. It
  decides exactly what gets built.
- **Architecture Decision Records** are a cross-cutting event, not a phase. They
  record why something was locked in, and they constrain future specs.

## Principles

1. **Files are truth.** Delete the desktop app and you lose pixels, nothing
   else. Every read is derivable from the `.flow/` trees.
2. **Git is the event log.** Frontmatter `status` is the current state; the
   commits that changed it are the transition history. No second ledger.
3. **One lineage per idea.** A pitch, the spec it becomes, and the decisions it
   emits all carry the same lineage id.
4. **Transitions are human gates.** Sub-agents draft artefacts and stop. A
   person commits the state change.
5. **One core, two lenses.** Indexing, parsing, id resolution, and write-routing
   live in a single Go package. CLI and desktop are thin clients over it.

## State machine

```
shaping -> betting -> specifying -> building -> cooldown -> archived
```

| From | To | Gate verb |
|------|-----|-----------|
| (none) | shaping | `flow shape <title>` |
| shaping | betting | `flow table --add <lineage>` |
| betting | specifying | `flow spec --from <lineage>` |
| specifying | building | `flow build --start <lineage>` |
| building | cooldown | `flow archive --cooldown <cycle>` |
| cooldown | archived | `flow archive --close <cycle>` |

ADRs run alongside all of these on their own lifecycle:
`proposed -> accepted -> superseded`.

## Install

Requires Go 1.26+.

```sh
go install github.com/juststeveking/flow/cmd/flow@latest
```

Or from a clone:

```sh
make cli        # -> bin/flow
```

## Usage

```sh
# Start a .flow/ tree in the current repo
flow init --id juststeveking/myproject

# Shape a pitch
flow shape "Passkey login" --appetite small-batch

# Put it on the betting table, then place the bet
flow table --add 2026-08-passkey-login

# Turn the bet into a spec, then lock it into building
flow spec --from 2026-08-passkey-login
flow build --start 2026-08-passkey-login

# Move a scope up the hill
flow build --spec 2026-08-passkey-login --hill scope-webauthn=0.6

# Record a decision
flow decide "WebAuthn library choice" --lineage 2026-08-passkey-login --reversibility one-way
flow decide --accept adr-0012-webauthn-library

# Read-only summary
flow status
```

Pass `-c` / `--commit` to any gate verb to commit the transition in the project
repo as part of the same command.

## Desktop overlay

`flow-desktop` is a [Wails v3](https://wails.io) window over an in-process HTTP
handler that aggregates every registered `.flow/` tree. It holds no truth of its
own; writes resolve back to exactly one project.

```sh
make desktop    # builds the React frontend, then bin/flow-desktop
```

The frontend build must run first — `cmd/flow-desktop/main.go` `go:embed`s
`frontend/dist`, which is not tracked in git. On a fresh clone,
`go build ./...` will fail on that package until you have run
`make desktop-frontend`. The `test`, `vet`, and `cli` targets deliberately
scope around it so the Go surface builds without Node.

## Layout

```
cmd/flow/            CLI entrypoint
cmd/flow-desktop/    Wails overlay + React frontend
internal/core/       Parsing, indexing, id resolution, write routing — the one model
internal/cli/        Cobra command surface over core
internal/desktop/    Workspace aggregation, HTTP API, project registry
internal/watch/      fsnotify watcher feeding live overlay updates
.flow/               This project's own pitches, specs, cycles, and decisions
```

flow runs on flow: `.flow/` in this repo is the real working tree for the
project, not a fixture.

## Development

```sh
make test       # go test ./...
make vet        # go vet ./...
gofmt -l ./internal ./cmd/flow ./cmd/flow-desktop
```

## Documents

- [PLAN.md](PLAN.md) — the full methodology, state machine, and architecture.
- [SCHEMA.md](SCHEMA.md) — the frozen frontmatter contract (schema version 1)
  that the parser implements against.
- [.claude/agents/README.md](.claude/agents/README.md) — the five sub-agents,
  each of which proposes and stops at a gate.

## Licence

MIT. See [LICENSE](LICENSE).
