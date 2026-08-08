package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Advisory locking for agents sharing one checkout. A lock is a file created with
// O_CREAT|O_EXCL, so exactly one caller can take it; the loser reports the holder
// and moves on. Locks are runtime-only and gitignored (.flow/locks/). This layer
// is advisory; the cross-checkout guarantee is the claim commit (see git.Push).

// DefaultLockTTL is how long a lock is honoured before it counts as stale and may
// be broken with a warning.
const DefaultLockTTL = 30 * time.Minute

// LocksDir is the runtime lock directory for a project root.
func LocksDir(root string) string { return filepath.Join(root, ".flow", "locks") }

func lockPath(root, id string) string { return filepath.Join(LocksDir(root), id) }

// Lock records who holds a task and since when.
type Lock struct {
	Holder string `yaml:"holder"`
	PID    int    `yaml:"pid"`
	Since  string `yaml:"since"` // RFC3339, UTC
}

// ErrLocked reports that a task is already held by a live, non-stale lock.
type ErrLocked struct{ Lock Lock }

func (e *ErrLocked) Error() string {
	return fmt.Sprintf("held by %s (pid %d) since %s", e.Lock.Holder, e.Lock.PID, e.Lock.Since)
}

// stale reports whether the lock is older than ttl. An unparseable or empty
// timestamp is deliberately NOT stale: it is the signature of a lock caught
// mid-write by a concurrent acquirer, so breaking on it would violate the
// exactly-one guarantee. A genuinely corrupt lock is cleared with --force.
func (l Lock) stale(now time.Time, ttl time.Duration) bool {
	since, err := time.Parse(time.RFC3339, l.Since)
	if err != nil {
		return false
	}
	return now.Sub(since) > ttl
}

// AcquireLock takes the lock for id on behalf of holder. It returns the lock it
// wrote; a non-nil broke names a stale or forced lock it replaced. If the task is
// held by a live lock and force is false, it returns *ErrLocked naming the holder.
func AcquireLock(root, id, holder string, ttl time.Duration, force bool) (lock Lock, broke *Lock, err error) {
	if err := os.MkdirAll(LocksDir(root), 0o755); err != nil {
		return Lock{}, nil, err
	}
	path := lockPath(root, id)
	mine := Lock{Holder: holder, PID: os.Getpid(), Since: time.Now().UTC().Format(time.RFC3339)}

	switch werr := writeLockExcl(path, mine); {
	case werr == nil:
		return mine, nil, nil
	case !errors.Is(werr, os.ErrExist):
		return Lock{}, nil, werr
	}

	// Held: inspect the current holder.
	cur, ok, rerr := ReadLock(root, id)
	if rerr != nil {
		return Lock{}, nil, rerr
	}
	if !ok {
		// Vanished between our create and read; try once more.
		if werr := writeLockExcl(path, mine); werr != nil {
			return Lock{}, nil, werr
		}
		return mine, nil, nil
	}
	if !force && !cur.stale(time.Now(), ttl) {
		return Lock{}, nil, &ErrLocked{Lock: cur}
	}
	// Break a stale or forced lock and take it.
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Lock{}, nil, err
	}
	if werr := writeLockExcl(path, mine); werr != nil {
		return Lock{}, nil, werr
	}
	prev := cur
	return mine, &prev, nil
}

// writeLockExcl creates the lock file exclusively and writes its contents. An
// existing file makes OpenFile fail with os.ErrExist, which is the whole point.
func writeLockExcl(path string, l Lock) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	b, mErr := yaml.Marshal(l)
	if mErr != nil {
		f.Close()
		return mErr
	}
	if _, wErr := f.Write(b); wErr != nil {
		f.Close()
		return wErr
	}
	return f.Close()
}

// ReadLock returns the lock held for id, or ok=false if none is held.
func ReadLock(root, id string) (Lock, bool, error) {
	b, err := os.ReadFile(lockPath(root, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Lock{}, false, nil
		}
		return Lock{}, false, err
	}
	var l Lock
	if err := yaml.Unmarshal(b, &l); err != nil {
		return Lock{}, false, err
	}
	return l, true, nil
}

// ReleaseLock drops the lock for id. Releasing an absent lock is not an error.
func ReleaseLock(root, id string) error {
	err := os.Remove(lockPath(root, id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
