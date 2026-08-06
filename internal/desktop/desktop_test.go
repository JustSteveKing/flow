package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

// scaffoldProject writes a minimal valid project with one shaping pitch and one
// active cycle, using the same lineage id regardless of project so tests can
// prove project-qualified identity does not collide.
func scaffoldProject(t *testing.T, root, projectID string) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, ".flow", "pitches"), 0o755))
	must(os.MkdirAll(filepath.Join(root, ".flow", "cycles"), 0o755))
	must(os.WriteFile(filepath.Join(root, ".flow", "manifest.yaml"),
		[]byte("version: 1\nproject: "+projectID+"\nproject_id: "+projectID+"\nbaseline:\n  mode: vendored\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, ".flow", "pitches", "2026-07-shared.md"),
		[]byte("---\nlineage: 2026-07-shared\ntype: pitch\nstatus: shaping\ntitle: Shared "+projectID+"\nappetite: big-batch\nproblem: p\ncreated: 2026-07-21\nupdated: 2026-07-21\n---\nbody\n"), 0o644))
	must(os.WriteFile(filepath.Join(root, ".flow", "cycles", "2026-C1.md"),
		[]byte("---\nid: 2026-C1\ntype: cycle\nstatus: active\nappetite_weeks: 6\ncapacity: 1\nstarts: 2026-07-21\nends: 2026-09-01\nbets: []\nshelved: []\n---\n"), 0o644))
}

func TestRegistryValidate(t *testing.T) {
	r := &Registry{Version: 1, Roots: []Root{
		{Path: "/a", ProjectID: "x/one"},
		{Path: "/b", ProjectID: "x/one"},
	}}
	if err := r.Validate(); err == nil {
		t.Fatal("duplicate project_id should be rejected")
	}
	r.Roots[1].ProjectID = "x/two"
	if err := r.Validate(); err != nil {
		t.Fatalf("valid registry rejected: %v", err)
	}
	r.Roots = append(r.Roots, Root{Path: "relative/path", ProjectID: "x/three"})
	if err := r.Validate(); err == nil {
		t.Fatal("relative path should be rejected")
	}
}

func TestWorkspaceAggregationAndRouting(t *testing.T) {
	base := t.TempDir()
	rootA := filepath.Join(base, "a")
	rootB := filepath.Join(base, "b")
	scaffoldProject(t, rootA, "acme/one")
	scaffoldProject(t, rootB, "acme/two")

	reg := &Registry{Version: 1, Roots: []Root{
		{Path: rootA, ProjectID: "acme/one"},
		{Path: rootB, ProjectID: "acme/two"},
	}}
	ws := NewWorkspace(reg, nil)
	if err := ws.Reload(); err != nil {
		t.Fatal(err)
	}

	// Both projects present; same local lineage does not collide.
	if got := len(ws.Projects()); got != 2 {
		t.Fatalf("want 2 projects, got %d", got)
	}

	// Empty betting table initially.
	if got := len(ws.BettingTable()); got != 0 {
		t.Fatalf("want empty table, got %d rows", got)
	}

	// Route a bet to project one only.
	if err := ws.PlaceBet("acme/one", "2026-07-shared"); err != nil {
		t.Fatalf("PlaceBet: %v", err)
	}
	table := ws.BettingTable()
	if len(table) != 1 {
		t.Fatalf("want 1 bet after routing, got %d: %+v", len(table), table)
	}
	if table[0].ProjectID != "acme/one" {
		t.Fatalf("bet routed to wrong project: %+v", table[0])
	}

	// Project two must be untouched on disk.
	bContent, _ := os.ReadFile(filepath.Join(rootB, ".flow", "cycles", "2026-C1.md"))
	if string(bContent) == "" || contains(bContent, "2026-07-shared\n      appetite") {
		t.Fatalf("write leaked into project two")
	}
	// Confirm project one file did get the bet.
	aContent, _ := os.ReadFile(filepath.Join(rootA, ".flow", "cycles", "2026-C1.md"))
	if !contains(aContent, "2026-07-shared") {
		t.Fatalf("bet not written to project one:\n%s", aContent)
	}

	// Capacity forcing function still bites through the overlay.
	if err := ws.PlaceBet("acme/one", "2026-07-shared"); err == nil {
		t.Fatal("second bet should be refused (already bet / capacity)")
	}
}

func contains(b []byte, s string) bool {
	return len(b) >= len(s) && indexOf(string(b), s) >= 0
}

func indexOf(hay, needle string) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
