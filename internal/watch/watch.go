// Package watch reflects external edits to `.flow/` trees into a re-index
// callback. It sits on top of internal/core (the CLI reuses the indexer without
// this; the desktop adds this loop). Every fsnotify hazard from PLAN.md section 7
// is handled here: directory watches, recursive walk-and-add, debounce and
// coalesce, ignore rules, the inotify budget, and the app-write echo guard.
package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches one or more `.flow/` subtrees and invokes OnChange after a
// debounce window whenever an external edit is seen. App-originated writes are
// swallowed so they do not echo back as external changes.
type Watcher struct {
	fsw      *fsnotify.Watcher
	debounce time.Duration
	onChange func()

	mu       sync.Mutex
	inflight map[string]int // absolute path -> pending self-writes to swallow
	dirty    bool
	timer    *time.Timer
}

// New creates a Watcher. onChange is called (coalesced) after external edits.
// debounce is the quiet period that collapses git event storms into one reindex.
func New(debounce time.Duration, onChange func()) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		fsw:      fsw,
		debounce: debounce,
		onChange: onChange,
		inflight: map[string]int{},
	}, nil
}

// RegisterWrite records that the app is about to write absPath, so the resulting
// fsnotify event is swallowed rather than treated as an external change. Wire
// this to core.NewStore's onWrite hook.
func (w *Watcher) RegisterWrite(absPath string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inflight[filepath.Clean(absPath)]++
}

// AddTree walks the `.flow/` subtree under root and adds a watch to every
// directory. fsnotify is non-recursive, so each directory is added explicitly.
// Watching only `.flow/` keeps the inotify watch count bounded by flow documents,
// not by the whole source tree.
func (w *Watcher) AddTree(root string) error {
	flow := filepath.Join(root, ".flow")
	return filepath.WalkDir(flow, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // tolerate a partially present tree
		}
		if d.IsDir() {
			if ignoredDir(d.Name()) {
				return filepath.SkipDir
			}
			_ = w.fsw.Add(path)
		}
		return nil
	})
}

// Run reads events until ctx is cancelled or the watcher is closed.
func (w *Watcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handle(ev)
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			// swallow watcher errors; a missed event is recovered on the next reindex
		}
	}
}

// Close stops the underlying watcher.
func (w *Watcher) Close() error {
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
	}
	w.mu.Unlock()
	return w.fsw.Close()
}

func (w *Watcher) handle(ev fsnotify.Event) {
	name := filepath.Clean(ev.Name)
	base := filepath.Base(name)

	// A newly created directory must be watched (fsnotify is non-recursive).
	if ev.Op&fsnotify.Create != 0 {
		if fi, err := os.Stat(name); err == nil && fi.IsDir() {
			if !ignoredDir(base) {
				_ = w.fsw.Add(name)
			}
			return
		}
	}

	// Ignore editor scratch, dotfiles, temp files, and non-markdown noise.
	if ignoredName(base) {
		return
	}
	if !strings.HasSuffix(base, ".md") && base != "manifest.yaml" {
		return
	}

	// Swallow app-originated writes so they do not echo as external edits.
	w.mu.Lock()
	if n := w.inflight[name]; n > 0 {
		w.inflight[name] = n - 1
		if w.inflight[name] == 0 {
			delete(w.inflight, name)
		}
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()

	w.markDirty()
}

// markDirty coalesces bursts: every relevant event resets the debounce timer, so
// onChange fires once after the tree goes quiet (handles atomic-save multi-events
// and git storms).
func (w *Watcher) markDirty() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.dirty = true
	if w.timer == nil {
		w.timer = time.AfterFunc(w.debounce, w.flush)
		return
	}
	w.timer.Reset(w.debounce)
}

func (w *Watcher) flush() {
	w.mu.Lock()
	fire := w.dirty
	w.dirty = false
	w.mu.Unlock()
	if fire && w.onChange != nil {
		w.onChange()
	}
}

// ignoredName filters editor scratch, dotfiles, and temp files. Kept in sync with
// the equivalent filter in core's indexer.
func ignoredName(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") {
		return true
	}
	if strings.HasSuffix(name, ".swp") || strings.HasSuffix(name, ".tmp") {
		return true
	}
	return false
}

// ignoredDir skips .git internals and dot-directories.
func ignoredDir(name string) bool {
	return name == ".git" || (strings.HasPrefix(name, ".") && name != ".flow")
}
