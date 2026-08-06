package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store performs write-routing and atomic writes. Every write resolves a document
// to exactly one project root and lands in a file; the store never holds truth
// (adr-0001). It is shared by the CLI and the desktop shell (adr-0002).
type Store struct {
	// onWrite is notified with the absolute path just before each write, so the
	// watcher (Phase 4) can record it in its in-flight set and swallow the echo.
	onWrite func(absPath string)
}

// NewStore returns a Store. onWrite may be nil (the CLI has no watcher).
func NewStore(onWrite func(absPath string)) *Store {
	return &Store{onWrite: onWrite}
}

// WriteOp is a proposed write, previewable before it is applied. This is the
// propose-then-write path every gate verb reuses.
type WriteOp struct {
	AbsPath string
	Before  []byte // current file bytes, nil if the file is new
	After   []byte
	New     bool
}

// Plan renders doc, routes it to its path under root, and captures the current
// bytes, without writing anything.
func (s *Store) Plan(root string, doc any) (WriteOp, error) {
	rel, err := RelPath(doc)
	if err != nil {
		return WriteOp{}, err
	}
	abs := filepath.Join(root, rel)
	after, err := Render(doc)
	if err != nil {
		return WriteOp{}, err
	}
	before, rerr := os.ReadFile(abs)
	op := WriteOp{AbsPath: abs, After: after}
	if rerr != nil {
		if !os.IsNotExist(rerr) {
			return WriteOp{}, rerr
		}
		op.New = true
	} else {
		op.Before = before
	}
	return op, nil
}

// NoChange reports whether applying the op would leave the file byte-identical.
func (o WriteOp) NoChange() bool {
	return !o.New && string(o.Before) == string(o.After)
}

// Apply writes the op atomically: a temp file in the target directory is renamed
// over the target, so a reader never sees a half-written file. The parent
// directory is created if absent (for example a new decision layer folder).
func (s *Store) Apply(op WriteOp) error {
	if op.NoChange() {
		return nil
	}
	dir := filepath.Dir(op.AbsPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if s.onWrite != nil {
		s.onWrite(op.AbsPath) // register before writing so the echo is swallowed
	}
	tmp, err := os.CreateTemp(dir, ".flow-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename
	if _, err := tmp.Write(op.After); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, op.AbsPath)
}

// DiffText renders a compact preview of an op for CLI and shell confirmation.
func (o WriteOp) DiffText() string {
	if o.New {
		return "create " + o.AbsPath + "\n" + prefixLines(string(o.After), "+ ")
	}
	if o.NoChange() {
		return "no change " + o.AbsPath
	}
	before := strings.Split(string(o.Before), "\n")
	after := strings.Split(string(o.After), "\n")
	lo := 0
	for lo < len(before) && lo < len(after) && before[lo] == after[lo] {
		lo++
	}
	hiB, hiA := len(before), len(after)
	for hiB > lo && hiA > lo && before[hiB-1] == after[hiA-1] {
		hiB--
		hiA--
	}
	var b strings.Builder
	fmt.Fprintf(&b, "modify %s\n", o.AbsPath)
	for _, l := range before[lo:hiB] {
		b.WriteString("- " + l + "\n")
	}
	for _, l := range after[lo:hiA] {
		b.WriteString("+ " + l + "\n")
	}
	return b.String()
}

func prefixLines(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n") + "\n"
}
