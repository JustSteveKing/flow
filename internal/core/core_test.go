package core

import (
	"os"
	"path/filepath"
	"testing"
)

// repoRoot is two levels up from internal/core.
const repoRoot = "../.."

func loadFixtureProject(t *testing.T) *ProjectIndex {
	t.Helper()
	idx, err := LoadProject(repoRoot)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	return idx
}

func fatalErrors(errs []DocError) []DocError {
	var out []DocError
	for _, e := range errs {
		if e.fatal() {
			out = append(out, e)
		}
	}
	return out
}

func TestLoadDogfoodProject(t *testing.T) {
	idx := loadFixtureProject(t)

	if fe := fatalErrors(idx.Errors); len(fe) != 0 {
		t.Fatalf("expected no fatal errors, got %v", fe)
	}
	if idx.Manifest == nil || idx.Manifest.ProjectID != "juststeveking/flow" {
		t.Fatalf("manifest project_id wrong: %+v", idx.Manifest)
	}
	if _, ok := idx.Pitches["2026-07-flow-genesis"]; !ok {
		t.Fatalf("genesis pitch not indexed; pitches=%v", keys(idx.Pitches))
	}
	if _, ok := idx.Cycles["2026-C1"]; !ok {
		t.Fatalf("cycle 2026-C1 not indexed")
	}
	// the dogfood tree carries the genesis ADRs; assert they resolve rather than
	// pinning an exact count that grows as flow dogfoods its own backlog.
	for _, id := range []string{"adr-0001-files-are-truth", "adr-0002-single-shared-core", "adr-0003-project-qualified-identity"} {
		if _, ok := idx.Decisions[id]; !ok {
			t.Fatalf("expected decision %s in index; have %v", id, keys(idx.Decisions))
		}
	}
	// a superseded record is excluded from the heads.
	for _, h := range idx.ADRHeads() {
		if h.Status == DecisionSuperseded {
			t.Fatalf("superseded ADR %s leaked into heads", h.ID)
		}
	}
}

func TestInvalidFixtures(t *testing.T) {
	dir := filepath.Join(repoRoot, "testdata", "invalid")

	cases := []struct {
		file          string
		wantTransient bool
		wantFatal     bool
	}{
		{"bad-status-pitch.md", false, true},
		{"truncated-frontmatter.md", true, false},
		{"hill-out-of-range-spec.md", false, true},
		{"broken-supersession.md", false, true},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(dir, c.file))
			if err != nil {
				t.Fatal(err)
			}
			_, derrs, transient := ParseDocument(filepath.Join(dir, c.file), content)
			if c.wantTransient {
				if transient == nil {
					t.Fatalf("expected transient error, got none")
				}
				return
			}
			if transient != nil {
				t.Fatalf("unexpected transient error: %v", transient)
			}
			if got := len(fatalErrors(derrs)) > 0; got != c.wantFatal {
				t.Fatalf("wantFatal=%v got errors=%v", c.wantFatal, derrs)
			}
		})
	}
}

func TestFrontmatterSplit(t *testing.T) {
	fm, body, err := splitFrontmatter([]byte("---\ntype: pitch\n---\nhello\nworld\n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(fm) != "type: pitch\n" {
		t.Fatalf("frontmatter %q", fm)
	}
	if body != "hello\nworld" {
		t.Fatalf("body %q", body)
	}
	if _, _, err := splitFrontmatter([]byte("no fence here\n")); err != ErrNoFrontmatter {
		t.Fatalf("want ErrNoFrontmatter, got %v", err)
	}
	if _, _, err := splitFrontmatter([]byte("---\ntype: pitch\nunclosed\n")); err != ErrUnterminatedFrontmatter {
		t.Fatalf("want ErrUnterminatedFrontmatter, got %v", err)
	}
}

func TestAggregateAndResolve(t *testing.T) {
	p := loadFixtureProject(t)
	idx, errs := NewIndex(p)
	if len(errs) != 0 {
		t.Fatalf("aggregate errors: %v", errs)
	}
	proj, local, ok := idx.Resolve("juststeveking/flow:2026-07-flow-genesis")
	if !ok || proj.Root != p.Root || local != "2026-07-flow-genesis" {
		t.Fatalf("resolve failed: ok=%v local=%q", ok, local)
	}
	if _, _, ok := idx.Resolve("nope"); ok {
		t.Fatalf("resolve of unqualified id should fail")
	}

	// the betting table aggregates the active cycle's bets; assert the genesis
	// lineage is present and carries a resolved title.
	var found bool
	for _, row := range idx.BettingTable() {
		if row.Lineage == "2026-07-flow-genesis" {
			found = true
			if row.Title == "" {
				t.Fatalf("betting row has no resolved title: %+v", row)
			}
		}
	}
	if !found {
		t.Fatalf("genesis bet not on betting table: %+v", idx.BettingTable())
	}
}

func TestSplitQualified(t *testing.T) {
	pid, local, ok := splitQualified("owner/name:adr-0001-x")
	if !ok || pid != "owner/name" || local != "adr-0001-x" {
		t.Fatalf("got %q %q %v", pid, local, ok)
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
