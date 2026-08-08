package desktop

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/juststeveking/flow/internal/core"
	"github.com/juststeveking/flow/internal/task"
	"github.com/juststeveking/flow/internal/watch"
)

// Workspace aggregates every registered project into one core.Index and routes
// writes back to a single project. It is the object the Wails shell binds to;
// the same object could back any other client (adr-0002).
type Workspace struct {
	registry *Registry

	mu       sync.RWMutex
	index    *core.Index
	tasks    map[string]*task.TaskSet  // by project id
	watchers map[string]*watch.Watcher // by project id

	debounce time.Duration
	onUpdate func() // notify the shell that the index changed
	now      func() string
}

// NewWorkspace builds a workspace over a registry. onUpdate may be nil.
func NewWorkspace(reg *Registry, onUpdate func()) *Workspace {
	return &Workspace{
		registry: reg,
		tasks:    map[string]*task.TaskSet{},
		watchers: map[string]*watch.Watcher{},
		debounce: 150 * time.Millisecond,
		onUpdate: onUpdate,
		now:      func() string { return time.Now().Format("2006-01-02") },
	}
}

// Reload re-indexes every registered root from files. The files are truth; this
// is a rebuildable projection (adr-0001).
func (w *Workspace) Reload() error {
	var projects []*core.ProjectIndex
	for _, root := range w.registry.Roots {
		p, err := core.LoadProject(root.Path)
		if err != nil {
			// a project that fails to load (for example no manifest) is skipped,
			// not fatal to the whole plane
			continue
		}
		projects = append(projects, p)
	}
	idx, _ := core.NewIndex(projects...)

	// Tasks live per project in .flow/tasks/; they are not part of the aggregated
	// core index. Load each registered root's set (files are truth, adr-0001).
	tasks := map[string]*task.TaskSet{}
	for _, root := range w.registry.Roots {
		if set, err := task.LoadTasks(root.Path); err == nil {
			tasks[root.ProjectID] = set
		}
	}

	w.mu.Lock()
	w.index = idx
	w.tasks = tasks
	w.mu.Unlock()
	return nil
}

// tasksSnapshot returns the current per-project task sets. Reload swaps the map
// wholesale, so the returned reference is safe to read without further locking.
func (w *Workspace) tasksSnapshot() map[string]*task.TaskSet {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.tasks
}

// rootFor resolves a project id to its filesystem root via the registry, so task
// routing works even for a project the core index dropped (for example a bad
// manifest) but whose task files are still present.
func (w *Workspace) rootFor(projectID string) (string, bool) {
	for _, r := range w.registry.Roots {
		if r.ProjectID == projectID {
			return r.Path, true
		}
	}
	return "", false
}

// TransitionTaskStatus moves a task to a new status through the same state machine
// the CLI uses, then regenerates the board index. An illegal transition (or a move
// to review with unticked criteria) is returned as an error, which the board turns
// into a rejected drop with a toast rather than a silent change.
func (w *Workspace) TransitionTaskStatus(projectID, id, toStatus string) error {
	root, ok := w.rootFor(projectID)
	if !ok {
		return fmt.Errorf("unknown project %q", projectID)
	}
	set, err := task.LoadTasks(root) // re-read: files are truth
	if err != nil {
		return err
	}
	t := set.Get(id)
	if t == nil {
		return fmt.Errorf("no task %q", id)
	}
	to := task.Status(toStatus)
	if !task.ValidStatus(to) {
		return fmt.Errorf("illegal status %q", toStatus)
	}
	if to == task.StatusReview {
		complete, err := task.ReviewComplete(t)
		if err != nil {
			return err
		}
		if !complete {
			return fmt.Errorf("cannot move %s to review: acceptance criteria incomplete", id)
		}
	}
	if err := task.TransitionTask(t, to); err != nil {
		return err
	}
	if err := t.Touch(w.now()); err != nil {
		return err
	}

	store := w.storeFor(projectID)
	taskOp, err := task.Plan(root, t)
	if err != nil {
		return err
	}
	if err := store.Apply(taskOp); err != nil {
		return err
	}
	idxOp, err := task.PlanIndex(root)
	if err != nil {
		return err
	}
	if err := store.Apply(idxOp); err != nil {
		return err
	}
	if core.IsRepo(root) {
		var paths []string
		if !taskOp.NoChange() {
			paths = append(paths, taskOp.AbsPath)
		}
		if !idxOp.NoChange() {
			paths = append(paths, idxOp.AbsPath)
		}
		if len(paths) > 0 {
			if err := core.Commit(root, "task status: "+id+" -> "+toStatus, paths...); err != nil {
				return err
			}
		}
	}

	_ = w.Reload()
	if w.onUpdate != nil {
		w.onUpdate()
	}
	return nil
}

// StartWatching adds a watcher per root and reloads on external edits. App writes
// are swallowed via each watcher's echo guard.
func (w *Workspace) StartWatching(ctx context.Context) error {
	for _, root := range w.registry.Roots {
		wa, err := watch.New(w.debounce, func() {
			_ = w.Reload()
			if w.onUpdate != nil {
				w.onUpdate()
			}
		})
		if err != nil {
			return err
		}
		if err := wa.AddTree(root.Path); err != nil {
			return err
		}
		w.watchers[root.ProjectID] = wa
		go wa.Run(ctx)
	}
	return nil
}

// storeFor returns a store whose writes are registered with the project's watcher,
// so app-originated writes do not echo back as external changes.
func (w *Workspace) storeFor(projectID string) *core.Store {
	if wa, ok := w.watchers[projectID]; ok {
		return core.NewStore(wa.RegisterWrite)
	}
	return core.NewStore(nil)
}

func (w *Workspace) snapshot() *core.Index {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.index
}

// --- Read views (aggregate across the whole plane) ---

// BettingTable is the unified betting table across all active cycles.
func (w *Workspace) BettingTable() []core.BettingTableRow {
	if idx := w.snapshot(); idx != nil {
		return idx.BettingTable()
	}
	return nil
}

// PortfolioHill is every building scope across all projects on one hill.
func (w *Workspace) PortfolioHill() []core.HillPoint {
	if idx := w.snapshot(); idx != nil {
		return idx.PortfolioHill()
	}
	return nil
}

// SupersessionGraph is every ADR supersession edge across all projects.
func (w *Workspace) SupersessionGraph() []core.SupersessionEdge {
	if idx := w.snapshot(); idx != nil {
		return idx.SupersessionGraph()
	}
	return nil
}

// ProjectSummary is a per-project overview for the registry panel.
type ProjectSummary struct {
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Root      string `json:"root"`
	Errors    int    `json:"errors"`
	Warnings  int    `json:"warnings"`
}

// Projects lists registered projects and their index health.
func (w *Workspace) Projects() []ProjectSummary {
	idx := w.snapshot()
	if idx == nil {
		return nil
	}
	var out []ProjectSummary
	for pid, p := range idx.Projects {
		var e, warn int
		for _, de := range p.Errors {
			if de.Severity == core.SevError {
				e++
			} else {
				warn++
			}
		}
		name := pid
		if p.Manifest != nil {
			name = p.Manifest.Project
		}
		out = append(out, ProjectSummary{ProjectID: pid, Name: name, Root: p.Root, Errors: e, Warnings: warn})
	}
	return out
}

// --- Write routing (always resolves to exactly one project) ---

// routed loads a fresh ProjectIndex for a project id (truth from files), runs a
// mutation that returns the documents to write, applies them atomically to that
// project's root, and commits them there. It never writes across two projects.
func (w *Workspace) routed(projectID string, mutate func(p *core.ProjectIndex) ([]any, string, error)) error {
	idx := w.snapshot()
	if idx == nil {
		return fmt.Errorf("workspace not loaded")
	}
	root, ok := idx.Root(projectID)
	if !ok {
		return fmt.Errorf("unknown project %q", projectID)
	}
	fresh, err := core.LoadProject(root) // re-read: files are truth
	if err != nil {
		return err
	}
	docs, msg, err := mutate(fresh)
	if err != nil {
		return err
	}
	store := w.storeFor(projectID)
	var paths []string
	for _, doc := range docs {
		op, err := store.Plan(root, doc)
		if err != nil {
			return err
		}
		if err := store.Apply(op); err != nil {
			return err
		}
		if !op.NoChange() {
			paths = append(paths, op.AbsPath)
		}
	}
	if len(paths) > 0 && core.IsRepo(root) {
		if err := core.Commit(root, msg, paths...); err != nil {
			return err
		}
	}
	_ = w.Reload()
	if w.onUpdate != nil {
		w.onUpdate()
	}
	return nil
}

// PlaceBet places a bet in a project's active cycle, enforcing capacity.
func (w *Workspace) PlaceBet(projectID, lineage string) error {
	return w.routed(projectID, func(p *core.ProjectIndex) ([]any, string, error) {
		cycle, err := singleActiveCycle(p)
		if err != nil {
			return nil, "", err
		}
		pitch := p.Pitches[lineage]
		if pitch == nil {
			return nil, "", fmt.Errorf("no pitch %q", lineage)
		}
		if err := core.TransitionPitch(pitch, core.PitchShaped); err != nil {
			return nil, "", err
		}
		if err := core.TransitionPitch(pitch, core.PitchBet); err != nil {
			return nil, "", err
		}
		if err := cycle.PlaceBet(lineage, pitch.Appetite, w.now()); err != nil {
			return nil, "", err
		}
		pitch.Updated = w.now()
		return []any{cycle, pitch}, "bet: " + lineage + " -> " + cycle.ID, nil
	})
}

// ShelveBet shelves a pitch in a project's active cycle.
func (w *Workspace) ShelveBet(projectID, lineage string) error {
	return w.routed(projectID, func(p *core.ProjectIndex) ([]any, string, error) {
		cycle, err := singleActiveCycle(p)
		if err != nil {
			return nil, "", err
		}
		pitch := p.Pitches[lineage]
		if pitch == nil {
			return nil, "", fmt.Errorf("no pitch %q", lineage)
		}
		if err := core.TransitionPitch(pitch, core.PitchShelved); err != nil {
			return nil, "", err
		}
		cycle.Shelve(lineage)
		pitch.Updated = w.now()
		return []any{cycle, pitch}, "shelve: " + lineage, nil
	})
}

// SetHill updates a scope's hill position in a building spec.
func (w *Workspace) SetHill(projectID, lineage, scopeID string, v float64) error {
	if v < 0 || v > 1 {
		return fmt.Errorf("hill must be in [0.0, 1.0]")
	}
	return w.routed(projectID, func(p *core.ProjectIndex) ([]any, string, error) {
		s := p.Specs[lineage]
		if s == nil {
			return nil, "", fmt.Errorf("no spec %q", lineage)
		}
		found := false
		for i := range s.Scopes {
			if s.Scopes[i].ID == scopeID {
				val := v
				s.Scopes[i].Hill = &val
				s.Scopes[i].Status = hillStatus(v)
				found = true
			}
		}
		if !found {
			return nil, "", fmt.Errorf("no scope %q", scopeID)
		}
		s.Updated = w.now()
		return []any{s}, fmt.Sprintf("hill: %s %s=%.2f", lineage, scopeID, v), nil
	})
}

func singleActiveCycle(p *core.ProjectIndex) (*core.Cycle, error) {
	var active []*core.Cycle
	for _, c := range p.Cycles {
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
		return nil, fmt.Errorf("ambiguous: %d active cycles", len(active))
	}
}

func hillStatus(v float64) core.ScopeStatus {
	switch {
	case v >= 1.0:
		return core.ScopeDone
	case v >= 0.5:
		return core.ScopeDownhill
	default:
		return core.ScopeUphill
	}
}
