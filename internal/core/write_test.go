package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripPitch(t *testing.T) {
	idx := loadFixtureProject(t)
	p := idx.Pitches["2026-07-flow-genesis"]
	if p == nil {
		t.Fatal("no genesis pitch")
	}
	rendered, err := Render(p)
	if err != nil {
		t.Fatal(err)
	}
	doc, derrs, transient := ParseDocument("mem://pitch", rendered)
	if transient != nil || fatal(derrs) {
		t.Fatalf("re-parse failed: transient=%v errs=%v", transient, derrs)
	}
	got := doc.(*Pitch)
	if got.Lineage != p.Lineage || got.Status != p.Status || got.Title != p.Title || got.Appetite != p.Appetite {
		t.Fatalf("round-trip mismatch: %+v vs %+v", got, p)
	}
	if len(got.AssumesDecisions) != len(p.AssumesDecisions) {
		t.Fatalf("assumes_decisions lost in round-trip")
	}
}

func TestPitchTransitions(t *testing.T) {
	p := &Pitch{Status: PitchShaping}
	if err := TransitionPitch(p, PitchBet); err == nil {
		t.Fatal("shaping -> bet should be illegal")
	}
	if err := TransitionPitch(p, PitchShaped); err != nil {
		t.Fatalf("shaping -> shaped should be legal: %v", err)
	}
	if err := TransitionPitch(p, PitchBet); err != nil {
		t.Fatalf("shaped -> bet should be legal: %v", err)
	}
	if err := TransitionPitch(p, PitchShaping); err != nil {
		t.Fatalf("bet -> shaping (circuit breaker) should be legal: %v", err)
	}
}

func TestSpecBuildGuard(t *testing.T) {
	s := &Spec{Status: SpecSpecifying, Scopes: []Scope{{ID: "a"}}}
	if err := TransitionSpec(s, SpecBuilding); err == nil {
		t.Fatal("building without locked interfaces must fail (Decision 1)")
	}
	s.Interfaces = []Interface{{Name: "X", Contract: "does X"}}
	if err := TransitionSpec(s, SpecBuilding); err != nil {
		t.Fatalf("building with interfaces should pass: %v", err)
	}
}

func TestSupersedeLinksChain(t *testing.T) {
	older := &Decision{ID: "adr-0001-old", Layer: LayerLocal, Status: DecisionAccepted}
	newer := &Decision{ID: "adr-0002-new", Layer: LayerLocal, Status: DecisionAccepted}
	if err := Supersede(older, newer); err != nil {
		t.Fatal(err)
	}
	if older.Status != DecisionSuperseded || older.SupersededBy == nil || *older.SupersededBy != newer.ID {
		t.Fatalf("older not linked: %+v", older)
	}
	if newer.Supersedes == nil || *newer.Supersedes != older.ID {
		t.Fatalf("newer not linked: %+v", newer)
	}

	// baseline must not be superseded by local's inverse: local superseded by baseline is illegal
	loc := &Decision{ID: "adr-0003-loc", Layer: LayerLocal, Status: DecisionAccepted}
	base := &Decision{ID: "adr-0004-base", Layer: LayerBaseline, Status: DecisionAccepted}
	if err := Supersede(loc, base); err == nil {
		t.Fatal("baseline superseding local must be rejected (Decision 4)")
	}
}

func TestStoreAtomicWrite(t *testing.T) {
	root := t.TempDir()
	var notified []string
	s := NewStore(func(p string) { notified = append(notified, p) })

	pitch := &Pitch{
		Lineage: "2026-07-write-test", Type: TypePitch, Status: PitchShaping,
		Title: "Write test", Appetite: BigBatch, Created: "2026-07-21", Updated: "2026-07-21",
		Body: "Body text.",
	}
	op, err := s.Plan(root, pitch)
	if err != nil {
		t.Fatal(err)
	}
	if !op.New {
		t.Fatal("expected a new-file op")
	}
	if err := s.Apply(op); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".flow", "pitches", "2026-07-write-test.md")
	if op.AbsPath != want {
		t.Fatalf("routed to %s want %s", op.AbsPath, want)
	}
	if len(notified) != 1 || notified[0] != want {
		t.Fatalf("onWrite not fired correctly: %v", notified)
	}

	// re-read and re-parse the written file
	content, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	doc, derrs, transient := ParseDocument(want, content)
	if transient != nil || fatal(derrs) {
		t.Fatalf("written file did not re-parse: %v %v", transient, derrs)
	}
	if doc.(*Pitch).Body != "Body text." {
		t.Fatalf("body not preserved: %q", doc.(*Pitch).Body)
	}

	// idempotent re-apply is a no-op
	op2, _ := s.Plan(root, doc.(*Pitch))
	if !op2.NoChange() {
		t.Fatalf("re-plan should be a no-op, got diff:\n%s", op2.DiffText())
	}
}
