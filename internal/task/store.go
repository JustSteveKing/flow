package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/juststeveking/flow/internal/core"
	"gopkg.in/yaml.v3"
)

// TasksDir returns the flat directory holding one file per task. There is no
// nesting and no per-status subdirectory: status lives in the frontmatter, not in
// the path, so a transition never moves a file.
func TasksDir(root string) string { return filepath.Join(root, ".flow", "tasks") }

// TaskFilename is the on-disk name for a task: T-{id}-{slug}.md. The id is
// authoritative; the slug is cosmetic, which is why tasks resolve by id and never
// by filename match.
func TaskFilename(t *Task) string {
	return t.ID + "-" + core.Slugify(t.Title) + ".md"
}

// isTaskFile reports whether a directory entry is a task document, excluding the
// generated index, the agent protocol, optional sidecar logs, and editor scratch.
func isTaskFile(name string) bool {
	if !strings.HasPrefix(name, "T-") || !strings.HasSuffix(name, ".md") {
		return false
	}
	if strings.HasSuffix(name, ".log.md") { // optional union-merge sidecar
		return false
	}
	return true
}

// TaskSet is a rebuildable projection of .flow/tasks/, keyed by id. Like the core
// index it is never authoritative; the files are (adr-0001). Reload to refresh.
type TaskSet struct {
	Root      string
	Tasks     map[string]*Task // by id
	Errors    []core.DocError
	Transient []string // half-written files to retry after the debounce window
}

// LoadTasks scans TasksDir and indexes every task by id. A missing directory is an
// empty set, not an error. A duplicate id is recorded as an error and the later
// file is dropped, so id resolution stays unambiguous.
func LoadTasks(root string) (*TaskSet, error) {
	s := &TaskSet{Root: root, Tasks: map[string]*Task{}}
	dir := TasksDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !isTaskFile(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, rerr := os.ReadFile(path)
		if rerr != nil {
			s.Errors = append(s.Errors, core.DocError{Path: path, Severity: core.SevError, Message: rerr.Error()})
			continue
		}
		t, derrs, transient := Parse(path, content)
		if transient != nil {
			s.Transient = append(s.Transient, path)
			continue
		}
		s.Errors = append(s.Errors, derrs...)
		if fatalErrs(derrs) {
			continue // dropped from the set (MUST violation)
		}
		if prev, ok := s.Tasks[t.ID]; ok {
			s.Errors = append(s.Errors, core.DocError{Path: path, Field: "id", Severity: core.SevError, Message: fmt.Sprintf("duplicate id %q (also %s)", t.ID, prev.Path)})
			continue
		}
		s.Tasks[t.ID] = t
	}
	return s, nil
}

func fatalErrs(errs []core.DocError) bool {
	for _, e := range errs {
		if e.Severity == core.SevError {
			return true
		}
	}
	return false
}

// Get resolves a task by its authoritative id.
func (s *TaskSet) Get(id string) *Task { return s.Tasks[id] }

// All returns every task in id order, for deterministic listing and indexing.
func (s *TaskSet) All() []*Task {
	out := make([]*Task, 0, len(s.Tasks))
	for _, id := range s.sortedIDs() {
		out = append(out, s.Tasks[id])
	}
	return out
}

func (s *TaskSet) sortedIDs() []string {
	ids := make([]string, 0, len(s.Tasks))
	for id := range s.Tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

var taskSeqRe = regexp.MustCompile(`^T-(\d+)$`)

func taskSeq(id string) int {
	m := taskSeqRe.FindStringSubmatch(id)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// NextID returns the next sequential task id, one past the highest existing
// number, zero-padded to three digits (T-014). Padding is cosmetic; taskSeq reads
// the number regardless of width.
func NextID(existing map[string]*Task) string {
	max := 0
	for id := range existing {
		if n := taskSeq(id); n > max {
			max = n
		}
	}
	return fmt.Sprintf("T-%03d", max+1)
}

// New scaffolds a task with empty sections and status todo. It builds the document
// bytes and re-parses them, so the returned Task carries the same verbatim src and
// byte offsets a loaded one would, and every later mutation splices consistently.
func New(id, title string, tags, depends []string, created string) (*Task, error) {
	seed := &Task{
		ID: id, Title: title, Status: StatusTodo,
		Depends: depends, Tags: tags, Created: created, Updated: created,
	}
	fm, err := yaml.Marshal(seed)
	if err != nil {
		return nil, err
	}
	src := "---\n" + string(fm) + "---\n\n## Context\n\n## Acceptance criteria\n\n## Log\n"
	t, derrs, transient := Parse("", []byte(src))
	if transient != nil {
		return nil, transient
	}
	if fatalErrs(derrs) {
		return nil, fmt.Errorf("scaffold invalid: %v", derrs)
	}
	return t, nil
}

// Plan builds an atomic write op for a task, routed to TasksDir. An existing task
// is written back to the file it was loaded from (its slug is fixed at creation);
// a new one lands at T-{id}-{slug}.md. It mirrors core.Store.Plan so callers reuse
// core.Store.Apply, DiffText, and the commit path.
func Plan(root string, t *Task) (core.WriteOp, error) {
	abs := t.Path
	if abs == "" {
		abs = filepath.Join(TasksDir(root), TaskFilename(t))
	}
	op := core.WriteOp{AbsPath: abs, After: t.Bytes()}
	before, rerr := os.ReadFile(abs)
	if rerr != nil {
		if !os.IsNotExist(rerr) {
			return core.WriteOp{}, rerr
		}
		op.New = true
	} else {
		op.Before = before
	}
	return op, nil
}
