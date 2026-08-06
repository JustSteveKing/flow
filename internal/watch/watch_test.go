package watch

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/juststeveking/flow/internal/core"
)

func waitFor(t *testing.T, cond func() bool, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

func TestWatcherExternalVsAppWrite(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".flow", "pitches"), 0o755); err != nil {
		t.Fatal(err)
	}

	var fires int32
	w, err := New(30*time.Millisecond, func() { atomic.AddInt32(&fires, 1) })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.AddTree(root); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	// External edit: should fire onChange.
	ext := filepath.Join(root, ".flow", "pitches", "external.md")
	if err := os.WriteFile(ext, []byte("---\ntype: pitch\n---\nhi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, func() bool { return atomic.LoadInt32(&fires) >= 1 }, time.Second) {
		t.Fatalf("external edit did not fire onChange")
	}

	before := atomic.LoadInt32(&fires)

	// App-originated write through the store: should be swallowed (no new fire).
	store := core.NewStore(w.RegisterWrite)
	pitch := &core.Pitch{
		Lineage: "2026-07-app-write", Type: core.TypePitch, Status: core.PitchShaping,
		Title: "App write", Appetite: core.BigBatch, Created: "2026-07-21", Updated: "2026-07-21",
	}
	op, err := store.Plan(root, pitch)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(op); err != nil {
		t.Fatal(err)
	}

	// Give the watcher longer than the debounce window to (not) fire.
	time.Sleep(150 * time.Millisecond)
	if got := atomic.LoadInt32(&fires); got != before {
		t.Fatalf("app write echoed as external change: fires went %d -> %d", before, got)
	}
}
