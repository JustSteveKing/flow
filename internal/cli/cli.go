// Package cli implements the flow command surface. Every mutating verb is a thin
// wrapper over internal/core: it loads the project, mutates a document, previews
// the write, applies it, and leaves (or makes) the git commit that records the
// state transition.
package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/juststeveking/flow/internal/core"
	"github.com/spf13/cobra"
)

var (
	flagCommit bool // -c/--commit: make the transition commit, not just write
)

// clock is overridable in tests; returns year-month and full ISO date.
var clock = func() (yearMonth, date string) {
	now := time.Now()
	return now.Format("2006-01"), now.Format("2006-01-02")
}

// Execute runs the root command.
func Execute() {
	if err := newRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "flow",
		Short:         "flow, a file-based, git-tracked SDLC operating system",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVarP(&flagCommit, "commit", "c", false, "commit the change in the project repo (records the transition)")
	root.AddCommand(
		cmdInit(),
		cmdShape(),
		cmdTable(),
		cmdSpec(),
		cmdDecide(),
		cmdBuild(),
		cmdArchive(),
		cmdStatus(),
	)
	return root
}

// mustRoot finds the project root from the working directory.
func mustRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, ok := core.FindRoot(cwd)
	if !ok {
		return "", fmt.Errorf("no flow project found (looked for .flow/manifest.yaml); run `flow init`")
	}
	return root, nil
}

func loadProject() (string, *core.ProjectIndex, error) {
	root, err := mustRoot()
	if err != nil {
		return "", nil, err
	}
	idx, err := core.LoadProject(root)
	if err != nil {
		return "", nil, err
	}
	return root, idx, nil
}

// write previews and applies a single-document write, and commits when asked.
// It honours the propose-then-write ethos: the diff is always printed first.
func write(root string, doc any, commitMsg string) error {
	s := core.NewStore(nil)
	op, err := s.Plan(root, doc)
	if err != nil {
		return err
	}
	fmt.Println(op.DiffText())
	if op.NoChange() {
		return nil
	}
	if err := s.Apply(op); err != nil {
		return err
	}
	if flagCommit {
		if !core.IsRepo(root) {
			return fmt.Errorf("wrote %s but --commit needs a git repo at %s", op.AbsPath, root)
		}
		if err := core.Commit(root, commitMsg, op.AbsPath); err != nil {
			return err
		}
		fmt.Println("committed:", commitMsg)
	} else {
		fmt.Println("wrote (not committed; re-run with --commit or commit yourself):", op.AbsPath)
	}
	return nil
}

// writeMany previews and applies several document writes as one transition.
func writeMany(root string, commitMsg string, docs ...any) error {
	s := core.NewStore(nil)
	var paths []string
	for _, doc := range docs {
		op, err := s.Plan(root, doc)
		if err != nil {
			return err
		}
		fmt.Println(op.DiffText())
		if op.NoChange() {
			continue
		}
		if err := s.Apply(op); err != nil {
			return err
		}
		paths = append(paths, op.AbsPath)
	}
	if len(paths) == 0 {
		return nil
	}
	if flagCommit {
		if !core.IsRepo(root) {
			return fmt.Errorf("wrote %d files but --commit needs a git repo at %s", len(paths), root)
		}
		if err := core.Commit(root, commitMsg, paths...); err != nil {
			return err
		}
		fmt.Println("committed:", commitMsg)
	} else {
		fmt.Printf("wrote %d files (not committed; re-run with --commit or commit yourself)\n", len(paths))
	}
	return nil
}

// activeCycle returns the single active cycle, or an error if none or many.
func activeCycle(idx *core.ProjectIndex) (*core.Cycle, error) {
	var active []*core.Cycle
	for _, c := range idx.Cycles {
		if c.Status == core.CycleActive {
			active = append(active, c)
		}
	}
	switch len(active) {
	case 0:
		return nil, fmt.Errorf("no active cycle")
	case 1:
		return active[0], nil
	default:
		return nil, fmt.Errorf("%d active cycles; ambiguous", len(active))
	}
}
