package task

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestConcurrentAcquireExactlyOneWins(t *testing.T) {
	root := t.TempDir()
	const n = 32
	var wins, locked int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range n {
		wg.Go(func() {
			<-start
			_, _, err := AcquireLock(root, "T-001", "agent:racer", DefaultLockTTL, false)
			switch {
			case err == nil:
				atomic.AddInt64(&wins, 1)
			case isLocked(err):
				atomic.AddInt64(&locked, 1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
	close(start)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("exactly one winner expected, got %d", wins)
	}
	if locked != n-1 {
		t.Fatalf("losers = %d, want %d", locked, n-1)
	}
}

func TestStaleLockBrokenRecordsPreviousHolder(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(LocksDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	// Plant a lock older than the TTL.
	stale := Lock{Holder: "agent:old", PID: 999, Since: time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}
	b, _ := yaml.Marshal(stale)
	if err := os.WriteFile(lockPath(root, "T-002"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	got, broke, err := AcquireLock(root, "T-002", "agent:new", 30*time.Minute, false)
	if err != nil {
		t.Fatalf("stale lock should be breakable: %v", err)
	}
	if broke == nil || broke.Holder != "agent:old" {
		t.Fatalf("break did not record previous holder: %+v", broke)
	}
	if got.Holder != "agent:new" {
		t.Fatalf("new holder = %q", got.Holder)
	}
}

func TestFreshLockNotStaleIsRefused(t *testing.T) {
	root := t.TempDir()
	if _, _, err := AcquireLock(root, "T-003", "agent:a", DefaultLockTTL, false); err != nil {
		t.Fatal(err)
	}
	_, _, err := AcquireLock(root, "T-003", "agent:b", DefaultLockTTL, false)
	if !isLocked(err) {
		t.Fatalf("a fresh lock must be refused, got %v", err)
	}
	var le *ErrLocked
	if errors.As(err, &le); le.Lock.Holder != "agent:a" {
		t.Fatalf("ErrLocked should name the holder, got %+v", le)
	}
}

func TestForceBreaksFreshLock(t *testing.T) {
	root := t.TempDir()
	if _, _, err := AcquireLock(root, "T-004", "agent:a", DefaultLockTTL, false); err != nil {
		t.Fatal(err)
	}
	_, broke, err := AcquireLock(root, "T-004", "agent:b", DefaultLockTTL, true)
	if err != nil {
		t.Fatalf("force should break: %v", err)
	}
	if broke == nil || broke.Holder != "agent:a" {
		t.Fatalf("force break should name previous holder: %+v", broke)
	}
}

func TestReleaseThenReacquire(t *testing.T) {
	root := t.TempDir()
	if _, _, err := AcquireLock(root, "T-005", "agent:a", DefaultLockTTL, false); err != nil {
		t.Fatal(err)
	}
	if err := ReleaseLock(root, "T-005"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := ReadLock(root, "T-005"); ok {
		t.Fatal("lock should be gone after release")
	}
	if _, _, err := AcquireLock(root, "T-005", "agent:b", DefaultLockTTL, false); err != nil {
		t.Fatalf("re-acquire after release should succeed: %v", err)
	}
	// Releasing an absent lock is not an error.
	if err := ReleaseLock(root, "T-404"); err != nil {
		t.Fatalf("releasing an absent lock should be a no-op: %v", err)
	}
}

func TestLockFileContents(t *testing.T) {
	root := t.TempDir()
	if _, _, err := AcquireLock(root, "T-006", "human:steve", DefaultLockTTL, false); err != nil {
		t.Fatal(err)
	}
	l, ok, err := ReadLock(root, "T-006")
	if err != nil || !ok {
		t.Fatalf("read lock: %v ok=%v", err, ok)
	}
	if l.Holder != "human:steve" || l.PID != os.Getpid() || l.Since == "" {
		t.Fatalf("lock contents wrong: %+v", l)
	}
	if _, err := os.Stat(filepath.Join(LocksDir(root), "T-006")); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
}

func isLocked(err error) bool {
	var le *ErrLocked
	return errors.As(err, &le)
}
