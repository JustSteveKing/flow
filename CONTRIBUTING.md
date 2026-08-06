# Contributing

flow runs on flow. The `.flow/` tree in this repository is the real working
tree for the project, so changes here go through the same gates the tool
enforces.

## Before you open a pull request

```sh
make check      # gofmt, go vet, go test -race
```

If you touched the overlay:

```sh
make desktop-frontend
```

## How work is organised

Substantial changes start as a pitch, not a patch. If you want to propose one:

1. `flow shape "<title>" --appetite small-batch` — write the problem, the
   appetite, the no-gos, and the rabbit holes. Open a PR with just the pitch.
2. If it is bet on, a spec follows and the scopes get cut there.

Small fixes — typos, obvious bugs, missing test cases — need none of this. Open
the PR directly.

Transitions between states (`shape`, `table --add`, `spec --from`,
`build --start`, `archive`) are human gates. Do not automate them, and do not
change a document's `status` by hand; use the CLI verb so the commit records
the transition.

## Conventions

- The frontmatter contract in [SCHEMA.md](SCHEMA.md) is frozen at version 1.
  A change to it is an ADR, not an edit.
- Domain logic lives in `internal/core` only. The CLI and the desktop overlay
  are thin clients; neither carries its own copy of the model (adr-0002).
- New behaviour needs a test in the package that owns it.
- Invalid-document fixtures go in `testdata/invalid/`.
