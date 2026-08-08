package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/juststeveking/flow/internal/core"
)

// writeTask scaffolds a task and writes it into root's .flow/tasks/ via the core
// store's atomic path, returning the task.
func writeTask(t *testing.T, root, id, title string, depends []string) *Task {
	t.Helper()
	tk, err := New(id, title, nil, depends, "2026-08-01")
	if err != nil {
		t.Fatalf("New(%s): %v", id, err)
	}
	op, err := Plan(root, tk)
	if err != nil {
		t.Fatalf("Plan(%s): %v", id, err)
	}
	if err := core.NewStore(nil).Apply(op); err != nil {
		t.Fatalf("Apply(%s): %v", id, err)
	}
	return tk
}

func TestNewScaffoldsParseableTodo(t *testing.T) {
	tk, err := New("T-001", "Rate limit the ingestion endpoint", []string{"api"}, []string{"T-000"}, "2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	if tk.Status != StatusTodo {
		t.Fatalf("status = %q, want todo", tk.Status)
	}
	body := string(tk.Body())
	for _, section := range []string{"## Context", "## Acceptance criteria", "## Log"} {
		if !strings.Contains(body, section) {
			t.Fatalf("scaffold missing %q:\n%s", section, body)
		}
	}
	// Re-parses cleanly, and a no-op write is byte identical.
	got, derrs, transient := Parse("mem://scaffold", tk.Bytes())
	if transient != nil || fatal(derrs) {
		t.Fatalf("scaffold does not re-parse: %v %v", transient, derrs)
	}
	if string(got.Bytes()) != string(tk.Bytes()) {
		t.Fatal("scaffold not stable on round-trip")
	}
}

func TestTaskFilename(t *testing.T) {
	tk, _ := New("T-014", "Rate limit the ingestion endpoint", nil, nil, "2026-08-01")
	if got := TaskFilename(tk); got != "T-014-rate-limit-the-ingestion-endpoint.md" {
		t.Fatalf("filename = %q", got)
	}
}

func TestNextIDAllocation(t *testing.T) {
	if got := NextID(map[string]*Task{}); got != "T-001" {
		t.Fatalf("empty set NextID = %q, want T-001", got)
	}
	set := map[string]*Task{
		"T-001": {ID: "T-001"},
		"T-014": {ID: "T-014"}, // gaps must not lower the next id
		"T-007": {ID: "T-007"},
	}
	if got := NextID(set); got != "T-015" {
		t.Fatalf("NextID = %q, want T-015", got)
	}
}

func TestLoadTasksIndexesByIDAndSkipsNonTasks(t *testing.T) {
	root := t.TempDir()
	writeTask(t, root, "T-001", "First", nil)
	writeTask(t, root, "T-002", "Second", []string{"T-001"})

	// Non-task files in the same directory must be ignored.
	dir := TasksDir(root)
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agent protocol"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T-001.log.md"), []byte("sidecar log"), 0o644); err != nil {
		t.Fatal(err)
	}

	set, err := LoadTasks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Tasks) != 2 {
		t.Fatalf("loaded %d tasks, want 2: %v", len(set.Tasks), set.Errors)
	}
	if set.Get("T-002").Depends[0] != "T-001" {
		t.Fatal("depends not loaded")
	}
	if len(set.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", set.Errors)
	}
}

func TestLoadTasksResolvesByIDNotFilename(t *testing.T) {
	root := t.TempDir()
	writeTask(t, root, "T-005", "Original title", nil)

	// Rename the file to a slug that no longer matches the title; id must still win.
	dir := TasksDir(root)
	old := filepath.Join(dir, "T-005-original-title.md")
	renamed := filepath.Join(dir, "T-005-totally-different-slug.md")
	if err := os.Rename(old, renamed); err != nil {
		t.Fatal(err)
	}

	set, err := LoadTasks(root)
	if err != nil {
		t.Fatal(err)
	}
	if set.Get("T-005") == nil {
		t.Fatal("retitled/renamed task not resolvable by id")
	}
}

func TestLoadTasksMissingDirIsEmpty(t *testing.T) {
	set, err := LoadTasks(t.TempDir())
	if err != nil {
		t.Fatalf("missing tasks dir should not error: %v", err)
	}
	if len(set.Tasks) != 0 {
		t.Fatal("expected empty set")
	}
}

func TestLoadTasksDuplicateID(t *testing.T) {
	root := t.TempDir()
	dir := TasksDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nid: T-009\ntitle: X\nstatus: todo\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n"
	if err := os.WriteFile(filepath.Join(dir, "T-009-a.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "T-009-b.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := LoadTasks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Tasks) != 1 {
		t.Fatalf("duplicate id should keep one task, got %d", len(set.Tasks))
	}
	if !fatal(set.Errors) {
		t.Fatal("expected a duplicate-id error")
	}
}
