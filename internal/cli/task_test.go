package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newProject lays down a minimal .flow/ so the task commands can find a root, and
// chdirs into it for the duration of the test.
func newProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	flow := filepath.Join(root, ".flow")
	if err := os.MkdirAll(filepath.Join(flow, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "version: 1\nproject: test\nproject_id: acme/test\nbaseline:\n    mode: vendored\nid_prefix: adr\n"
	if err := os.WriteFile(filepath.Join(flow, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	return root
}

// runFlow executes the root command with args, capturing stdout. A fresh root per
// call resets the flag globals to their defaults.
func runFlow(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root := newRoot()
	root.SetArgs(args)
	err := root.Execute()

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), err
}

func TestTaskNewAndListJSON(t *testing.T) {
	newProject(t)

	out, err := runFlow(t, "task", "new", "First task", "--tag", "api")
	if err != nil {
		t.Fatalf("task new: %v", err)
	}
	if !strings.Contains(out, "T-001") {
		t.Fatalf("new did not print id: %q", out)
	}

	out, err = runFlow(t, "task", "list", "--json")
	if err != nil {
		t.Fatalf("task list: %v", err)
	}
	var views []taskView
	if err := json.Unmarshal([]byte(out), &views); err != nil {
		t.Fatalf("list --json not valid JSON: %v\n%s", err, out)
	}
	if len(views) != 1 || views[0].ID != "T-001" || views[0].Status != "todo" {
		t.Fatalf("unexpected list: %+v", views)
	}
	if len(views[0].Tags) != 1 || views[0].Tags[0] != "api" {
		t.Fatalf("tag not recorded: %+v", views[0])
	}
}

func TestTaskNextAndClaimFlow(t *testing.T) {
	newProject(t)
	if _, err := runFlow(t, "task", "new", "First"); err != nil {
		t.Fatal(err)
	}
	if _, err := runFlow(t, "task", "new", "Second", "--depends", "T-001"); err != nil {
		t.Fatal(err)
	}

	// T-001 is the only ready task (T-002 depends on it).
	out, err := runFlow(t, "task", "next", "--json")
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var v taskView
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("next --json: %v\n%s", err, out)
	}
	if v.ID != "T-001" {
		t.Fatalf("next = %s, want T-001", v.ID)
	}

	// Claiming T-002 must fail: its dependency is not done.
	if _, err := runFlow(t, "task", "claim", "T-002", "--as", "agent:claude"); err == nil {
		t.Fatal("expected claim of T-002 to fail on unfinished dependency")
	}

	// Claim T-001, then nothing is ready (T-001 doing, T-002 still blocked).
	if _, err := runFlow(t, "task", "claim", "T-001", "--as", "agent:claude"); err != nil {
		t.Fatalf("claim T-001: %v", err)
	}
	out, err = runFlow(t, "task", "show", "T-001", "--json")
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal([]byte(out), &v)
	if v.Status != "doing" || v.Assignee != "agent:claude" {
		t.Fatalf("after claim: status=%s assignee=%s", v.Status, v.Assignee)
	}
	if _, err := runFlow(t, "task", "next"); err == nil {
		t.Fatal("expected non-zero exit when no task is ready")
	}
}

func TestTaskStatusRejectsDoingToDone(t *testing.T) {
	newProject(t)
	if _, err := runFlow(t, "task", "new", "Work"); err != nil {
		t.Fatal(err)
	}
	if _, err := runFlow(t, "task", "claim", "T-001", "--as", "agent:claude"); err != nil {
		t.Fatal(err)
	}
	// doing -> done must be rejected by the state machine.
	if _, err := runFlow(t, "task", "status", "T-001", "done"); err == nil {
		t.Fatal("doing -> done should be rejected")
	}
	// blocked without a reason must be rejected.
	if _, err := runFlow(t, "task", "status", "T-001", "blocked"); err == nil {
		t.Fatal("blocked without a reason should be rejected")
	}
	// blocked with a reason succeeds and logs it.
	if _, err := runFlow(t, "task", "status", "T-001", "blocked", "waiting on upstream"); err != nil {
		t.Fatalf("blocked with reason: %v", err)
	}
	out, err := runFlow(t, "task", "show", "T-001")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "blocked: waiting on upstream") {
		t.Fatalf("blocked reason not logged:\n%s", out)
	}
}

func TestNewAutoRegeneratesIndex(t *testing.T) {
	root := newProject(t)
	if _, err := runFlow(t, "task", "new", "Rate limit ingestion"); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(root, ".flow", "TASKS.md"))
	if err != nil {
		t.Fatalf("TASKS.md not generated after new: %v", err)
	}
	if !strings.Contains(string(idx), "## Todo") || !strings.Contains(string(idx), "T-001") {
		t.Fatalf("index does not list the new task:\n%s", idx)
	}

	// Claiming moves it into Doing on the next regeneration.
	if _, err := runFlow(t, "task", "claim", "T-001", "--as", "agent:claude"); err != nil {
		t.Fatal(err)
	}
	idx, _ = os.ReadFile(filepath.Join(root, ".flow", "TASKS.md"))
	if !strings.Contains(string(idx), "## Doing") || !strings.Contains(string(idx), "@claude") {
		t.Fatalf("index not updated after claim:\n%s", idx)
	}
}

func TestSyncIsIdempotent(t *testing.T) {
	root := newProject(t)
	if _, err := runFlow(t, "task", "new", "A task"); err != nil {
		t.Fatal(err)
	}
	if _, err := runFlow(t, "task", "sync"); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(root, ".flow", "TASKS.md"))
	if _, err := runFlow(t, "task", "sync"); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(root, ".flow", "TASKS.md"))
	if string(first) != string(second) {
		t.Fatal("sync must be idempotent")
	}
}

func TestInitScaffoldsTaskArtifacts(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if _, err := runFlow(t, "init", "--id", "acme/demo"); err != nil {
		t.Fatalf("init: %v", err)
	}
	agents, err := os.ReadFile(filepath.Join(root, ".flow", "tasks", "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md not scaffolded: %v", err)
	}
	if !strings.Contains(string(agents), "Never set status to") {
		t.Fatalf("AGENTS.md missing the protocol:\n%s", agents)
	}
	attrs, err := os.ReadFile(filepath.Join(root, ".gitattributes"))
	if err != nil {
		t.Fatalf(".gitattributes not written: %v", err)
	}
	if !strings.Contains(string(attrs), ".flow/TASKS.md -diff") {
		t.Fatalf(".gitattributes missing the -diff rule:\n%s", attrs)
	}
}
