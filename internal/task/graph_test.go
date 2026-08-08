package task

import "testing"

// setOf builds a TaskSet from inline tasks for graph tests.
func setOf(tasks ...*Task) *TaskSet {
	s := &TaskSet{Tasks: map[string]*Task{}}
	for _, t := range tasks {
		s.Tasks[t.ID] = t
	}
	return s
}

func tk(id string, status Status, depends ...string) *Task {
	return &Task{ID: id, Status: status, Depends: depends}
}

func TestReadyRequiresTodoAndDoneDeps(t *testing.T) {
	s := setOf(
		tk("T-001", StatusDone),
		tk("T-002", StatusTodo, "T-001"), // ready: dep done
		tk("T-003", StatusTodo, "T-002"), // blocked: dep not done
		tk("T-004", StatusDoing),         // not todo
		tk("T-005", StatusTodo),          // ready: no deps
	)
	ready, err := s.Ready()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range ready {
		got[r.ID] = true
	}
	if !got["T-002"] || !got["T-005"] {
		t.Fatalf("expected T-002 and T-005 ready, got %v", got)
	}
	if got["T-003"] || got["T-004"] || got["T-001"] {
		t.Fatalf("unexpected task in ready set: %v", got)
	}
}

func TestReadyIsIDOrdered(t *testing.T) {
	s := setOf(tk("T-005", StatusTodo), tk("T-002", StatusTodo), tk("T-009", StatusTodo))
	ready, err := s.Ready()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"T-002", "T-005", "T-009"}
	for i, r := range ready {
		if r.ID != want[i] {
			t.Fatalf("ready[%d] = %s, want %s", i, r.ID, want[i])
		}
	}
}

func TestBlockersNamesMissingAndUnfinished(t *testing.T) {
	s := setOf(
		tk("T-001", StatusReview),
		tk("T-002", StatusTodo, "T-001", "T-099"), // T-099 missing, T-001 not done
	)
	b := s.Blockers("T-002")
	if len(b) != 2 {
		t.Fatalf("blockers = %v, want two", b)
	}
}

func TestNextPicksLowestReadyID(t *testing.T) {
	s := setOf(tk("T-003", StatusTodo), tk("T-001", StatusDoing), tk("T-002", StatusTodo))
	n, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}
	if n == nil || n.ID != "T-002" {
		t.Fatalf("next = %v, want T-002", n)
	}
}

func TestNextNoneReady(t *testing.T) {
	s := setOf(tk("T-001", StatusDoing), tk("T-002", StatusDone))
	n, err := s.Next()
	if err != nil {
		t.Fatal(err)
	}
	if n != nil {
		t.Fatalf("expected no ready task, got %v", n)
	}
}

func TestCycleDetectionFailsNotLoops(t *testing.T) {
	s := setOf(
		tk("T-001", StatusTodo, "T-002"),
		tk("T-002", StatusTodo, "T-003"),
		tk("T-003", StatusTodo, "T-001"),
	)
	if err := s.DetectCycle(); err == nil {
		t.Fatal("expected a cycle error")
	}
	if _, err := s.Ready(); err == nil {
		t.Fatal("Ready must fail on a cyclic graph")
	}
}

func TestSelfDependencyIsACycle(t *testing.T) {
	s := setOf(tk("T-001", StatusTodo, "T-001"))
	if err := s.DetectCycle(); err == nil {
		t.Fatal("a task depending on itself is a cycle")
	}
}

func TestAcyclicPasses(t *testing.T) {
	s := setOf(
		tk("T-001", StatusDone),
		tk("T-002", StatusDone, "T-001"),
		tk("T-003", StatusTodo, "T-001", "T-002"),
	)
	if err := s.DetectCycle(); err != nil {
		t.Fatalf("acyclic graph flagged: %v", err)
	}
}
