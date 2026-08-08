// Package task is the task-management subsystem: one markdown file per task in a
// flat .flow/tasks/ directory, parsed and mutated without ever round-tripping the
// human-authored body through a markdown renderer.
//
// It is a domain package in its own right, a sibling of internal/core rather than
// a client of it (adr-0002 keeps model logic out of the CLI and desktop shells,
// which stay thin over this package). It reuses core's frontmatter framing and
// error vocabulary so the two subsystems read and report alike.
package task

import (
	"regexp"

	"github.com/juststeveking/flow/internal/core"
)

// Status is the machine field that drives the board. It is deliberately separate
// from the acceptance-criteria checkboxes in the body: status reports where the
// work sits, the checkboxes report how far it has got. An agent can tick a box to
// show partial progress without ever claiming the whole task is complete.
type Status string

const (
	StatusTodo    Status = "todo"
	StatusDoing   Status = "doing"
	StatusBlocked Status = "blocked"
	StatusReview  Status = "review"
	StatusDone    Status = "done"
	StatusDropped Status = "dropped"
)

// statuses is the membership set for validation, mirroring core's enum pattern.
var statuses = map[Status]bool{
	StatusTodo: true, StatusDoing: true, StatusBlocked: true,
	StatusReview: true, StatusDone: true, StatusDropped: true,
}

// ValidStatus reports whether s is one of the six legal statuses.
func ValidStatus(s Status) bool { return statuses[s] }

// Identifier shapes. The id is authoritative and the slug in the filename is
// cosmetic, so tasks resolve by id, never by filename match.
var (
	taskIDRe   = regexp.MustCompile(`^T-\d+$`)
	assigneeRe = regexp.MustCompile(`^(agent|human):\S+$`)
)

// ValidTaskID reports whether s is a well-formed task id (T- followed by digits).
func ValidTaskID(s string) bool { return taskIDRe.MatchString(s) }

// ValidAssignee reports whether s is agent:{name} or human:{name}. An empty
// assignee is legal (unassigned) but is not this function's concern.
func ValidAssignee(s string) bool { return assigneeRe.MatchString(s) }

// Task is one task document. The frontmatter fields are the machine-legible
// header; Body (via Body) is the verbatim human-authored markdown. Extra captures
// any unknown frontmatter keys so a re-serialisation preserves them, matching the
// store's unknown-field-preserving contract.
type Task struct {
	ID       string         `yaml:"id"`
	Title    string         `yaml:"title"`
	Status   Status         `yaml:"status"`
	Assignee string         `yaml:"assignee,omitempty"`
	Depends  []string       `yaml:"depends,omitempty"`
	Tags     []string       `yaml:"tags,omitempty"`
	Created  string         `yaml:"created"`
	Updated  string         `yaml:"updated"`
	Extra    map[string]any `yaml:",inline"`

	Path string `yaml:"-"` // source path, set by the loader

	// src holds the whole file verbatim; every mutation is a surgical splice into
	// it. Retaining it (rather than re-rendering from the struct) is what makes a
	// no-op write byte-identical and keeps the body free of renderer reformatting.
	src []byte
	// Byte spans into src. fmStart..fmEnd is the inner YAML (between the fences);
	// bodyStart is where the markdown body begins.
	fmStart, fmEnd, bodyStart int
}

// Body returns the verbatim markdown body (everything after the frontmatter).
func (t *Task) Body() []byte { return t.src[t.bodyStart:] }

// Bytes returns the current serialised document. Immediately after Parse, with no
// mutation applied, this equals the input byte for byte.
func (t *Task) Bytes() []byte { return t.src }

// validate accumulates MUST/SHOULD problems into DocErrors, following core's
// severity model: a MUST violation would drop the task from an index.
func (t *Task) validate() []core.DocError {
	b := &errbag{path: t.Path}
	req(b, "id", t.ID)
	if t.ID != "" && !ValidTaskID(t.ID) {
		b.errf("id", "malformed task id %q (want T-<number>)", t.ID)
	}
	req(b, "title", t.Title)
	if t.Status == "" {
		b.errf("status", "required")
	} else if !ValidStatus(t.Status) {
		b.errf("status", "illegal status %q", t.Status)
	}
	if t.Assignee != "" && !ValidAssignee(t.Assignee) {
		b.errf("assignee", "malformed assignee %q (want agent:name or human:name)", t.Assignee)
	}
	for _, id := range t.Depends {
		if !ValidTaskID(id) {
			b.errf("depends", "malformed task id %q", id)
		}
	}
	req(b, "created", t.Created)
	if t.Created != "" && !core.ValidDate(t.Created) {
		b.errf("created", "malformed date %q", t.Created)
	}
	req(b, "updated", t.Updated)
	if t.Updated != "" && !core.ValidDate(t.Updated) {
		b.errf("updated", "malformed date %q", t.Updated)
	}
	return b.errs
}
