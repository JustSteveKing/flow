---
id: adr-0008-one-file-per-task
type: decision
layer: local
status: proposed
reversibility: reversible
title: One markdown file per task, not a single tasks.md
supersedes: null
superseded_by: null
decided: 2026-08-08
---

## Context

The task subsystem needs a storage format that three consumers share: AI agents
that parse and mutate tasks programmatically, humans reading them in an editor and
on GitHub, and git, which must produce a history that bisects and reviews cleanly.

The obvious alternative is a single `tasks.md` (or `tasks.yaml`) holding every
task. It is one file to open and trivially lists everything at once.

## Decision

Each task is its own file in a flat `.flow/tasks/` directory, named
`T-{id}-{slug}.md`, with a YAML frontmatter header and a verbatim markdown body.
The id is authoritative and the slug is cosmetic; tasks resolve by id, never by
filename match. `.flow/TASKS.md` is a generated, derived index (adr-0001), never
hand-edited.

## Consequences

- **Concurrency.** Two agents mutating two different tasks touch two different
  files, so their writes never contend. A single `tasks.md` would serialise every
  edit and turn routine parallel work into frontmatter merge conflicts — and a
  conflict in a shared header can produce two `status:` lines, after which every
  parse fails. Per-file storage keeps the blast radius of a conflict to one task.
- **Locking and claim.** Advisory per-task lockfiles (`.flow/locks/T-014`) and the
  single-file claim commit both become natural: the unit of locking and of the
  commit is the file. The claim commit stays minimal and reviewable.
- **Git history.** One file per task yields a per-task history that bisects and
  `git log --follow`s. A monolithic file interleaves unrelated changes into every
  commit and every blame line.
- **Cost.** Listing the board means scanning a directory rather than reading one
  file, so `flow task sync` regenerates `TASKS.md`; `.flow/TASKS.md -diff` keeps
  the generated index out of reviews. An optional union-merge sidecar
  (`.flow/tasks/T-014.log.md`) can absorb concurrent Log appends — but union merge
  must never be applied to the task file itself, or a conflict would duplicate the
  `status:` line and break parsing.
