package core

import (
	"fmt"
	"os/exec"
	"strings"
)

// Git wraps the minimal git operations flow needs. Git history is the transition
// event log (adr-0001): each gate verb stages the changed files and commits in
// the project's own repository.

// IsRepo reports whether root is inside a git work tree.
func IsRepo(root string) bool {
	cmd := exec.Command("git", "-C", root, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

// Commit stages the given paths (relative to root or absolute) and commits them
// in root's repository with message. A commit is the record of a state transition.
func Commit(root, message string, paths ...string) error {
	if len(paths) == 0 {
		return fmt.Errorf("nothing to commit")
	}
	addArgs := append([]string{"-C", root, "add", "--"}, paths...)
	if out, err := exec.Command("git", addArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %v: %s", err, out)
	}
	out, err := exec.Command("git", "-C", root, "commit", "-m", message).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit: %v: %s", err, out)
	}
	return nil
}

// Root returns the filesystem root for a project id in the aggregated index.
func (i *Index) Root(projectID string) (string, bool) {
	p, ok := i.Projects[projectID]
	if !ok {
		return "", false
	}
	return p.Root, true
}
