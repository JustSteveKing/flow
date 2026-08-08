package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// git runs a git command in dir, failing the test on error.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func configIdentity(t *testing.T, dir string) {
	t.Helper()
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "Test")
}

// TestPushRejectionSurfaces proves the claim-commit concurrency guarantee: two
// checkouts share a remote; the one that pushes second is rejected, which the task
// layer reads as "someone else claimed first, reselect".
func TestPushRejectionSurfaces(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	base := t.TempDir()
	bare := filepath.Join(base, "remote.git")
	git(t, base, "init", "--bare", "-b", "main", bare)

	// Clone A establishes main on the remote.
	a := filepath.Join(base, "a")
	git(t, base, "clone", bare, a)
	configIdentity(t, a)
	if err := os.WriteFile(filepath.Join(a, "seed"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, a, "add", ".")
	git(t, a, "commit", "-m", "seed")
	git(t, a, "push", "-u", "origin", "main")

	// Clone B starts from the same point.
	b := filepath.Join(base, "b")
	git(t, base, "clone", bare, b)
	configIdentity(t, b)

	// A advances and pushes (fast-forward, fine).
	if err := os.WriteFile(filepath.Join(a, "seed"), []byte("2"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, a, "commit", "-am", "a advances")
	if err := Push(a); err != nil {
		t.Fatalf("A push should succeed: %v", err)
	}

	// B commits on the now-stale main and pushes: must be rejected.
	if err := os.WriteFile(filepath.Join(b, "seed"), []byte("3"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, b, "commit", "-am", "b advances")
	if err := Push(b); err == nil {
		t.Fatal("B push should be rejected (non-fast-forward)")
	}
}
