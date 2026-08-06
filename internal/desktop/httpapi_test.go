package desktop

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPAPISnapshotAndBet(t *testing.T) {
	base := t.TempDir()
	rootA := filepath.Join(base, "a")
	scaffoldProject(t, rootA, "acme/one")

	reg := &Registry{Version: 1, Roots: []Root{{Path: rootA, ProjectID: "acme/one"}}}
	ws := NewWorkspace(reg, nil)
	if err := ws.Reload(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(ws.Handler(nil))
	defer srv.Close()

	// snapshot: empty betting table, one project
	var snap Snapshot
	getJSON(t, srv.URL+"/api/snapshot", &snap)
	if len(snap.Projects) != 1 || len(snap.BettingTable) != 0 {
		t.Fatalf("unexpected initial snapshot: %+v", snap)
	}

	// place a bet
	if code := postJSON(t, srv.URL+"/api/bet", `{"projectId":"acme/one","lineage":"2026-07-shared"}`); code != 200 {
		t.Fatalf("bet POST returned %d", code)
	}

	getJSON(t, srv.URL+"/api/snapshot", &snap)
	if len(snap.BettingTable) != 1 || snap.BettingTable[0].ProjectID != "acme/one" {
		t.Fatalf("bet not reflected: %+v", snap.BettingTable)
	}

	// second bet must be refused (capacity/already-bet) with a non-200
	if code := postJSON(t, srv.URL+"/api/bet", `{"projectId":"acme/one","lineage":"2026-07-shared"}`); code == 200 {
		t.Fatal("second bet should be refused")
	}

	// project detail endpoint returns the full per-project model
	var detail ProjectDetail
	getJSON(t, srv.URL+"/api/project?id=acme/one", &detail)
	if detail.ProjectID != "acme/one" {
		t.Fatalf("detail project id wrong: %+v", detail)
	}
	if len(detail.Cycles) != 1 || len(detail.Pitches) != 1 {
		t.Fatalf("detail missing cycles/pitches: %+v", detail)
	}
	if len(detail.Cycles[0].Bets) != 1 {
		t.Fatalf("detail cycle should show the placed bet: %+v", detail.Cycles[0])
	}

	// doc source endpoint returns the raw on-disk markdown
	var doc DocView
	getJSON(t, srv.URL+"/api/doc?project=acme/one&kind=pitch&id=2026-07-shared", &doc)
	if doc.Path == "" || !strings.Contains(doc.Content, "lineage: 2026-07-shared") {
		t.Fatalf("doc source wrong: %+v", doc)
	}
	// frontmatter split into fields, body separated
	var hasLineage bool
	for _, f := range doc.Frontmatter {
		if f.Key == "lineage" && f.Value == "2026-07-shared" {
			hasLineage = true
		}
	}
	if !hasLineage {
		t.Fatalf("frontmatter fields missing lineage: %+v", doc.Frontmatter)
	}
	if !strings.Contains(doc.Body, "body") {
		t.Fatalf("body not separated: %q", doc.Body)
	}
	// unknown doc is a 404
	if r, _ := http.Get(srv.URL + "/api/doc?project=acme/one&kind=spec&id=nope"); r.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown doc want 404, got %d", r.StatusCode)
	}

	// unknown project is a 404
	resp, err := http.Get(srv.URL + "/api/project?id=nope/nope")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown project want 404, got %d", resp.StatusCode)
	}
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func postJSON(t *testing.T, url, body string) int {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}
