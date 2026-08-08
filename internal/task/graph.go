package task

import (
	"fmt"
	"strings"
)

// The dependency graph is built from the `depends` lists. A task is ready when its
// status is todo and every dependency exists and is done. Cycles are detected up
// front and reported, never looped over.

// Blockers returns the dependency ids of a task that are not yet satisfied: any
// that are missing from the set or whose status is not done. An empty result means
// every dependency is done.
func (s *TaskSet) Blockers(id string) []string {
	t := s.Tasks[id]
	if t == nil {
		return nil
	}
	var out []string
	for _, dep := range t.Depends {
		d := s.Tasks[dep]
		if d == nil || d.Status != StatusDone {
			out = append(out, dep)
		}
	}
	return out
}

// DetectCycle reports the first dependency cycle it finds, naming the path, or nil
// if the graph is acyclic. Traversal is deterministic (ids visited in order) so the
// reported cycle is stable. Edges to unknown ids are ignored here; Blockers surfaces
// those as unsatisfied dependencies instead.
func (s *TaskSet) DetectCycle() error {
	const (
		white = 0
		grey  = 1
		black = 2
	)
	colour := map[string]int{}
	var stack []string
	var walk func(id string) error
	walk = func(id string) error {
		colour[id] = grey
		stack = append(stack, id)
		if t := s.Tasks[id]; t != nil {
			for _, dep := range t.Depends {
				if _, known := s.Tasks[dep]; !known {
					continue
				}
				switch colour[dep] {
				case grey:
					return fmt.Errorf("dependency cycle: %s", strings.Join(append(stack, dep), " -> "))
				case white:
					if err := walk(dep); err != nil {
						return err
					}
				}
			}
		}
		colour[id] = black
		stack = stack[:len(stack)-1]
		return nil
	}
	for _, id := range s.sortedIDs() {
		if colour[id] == white {
			if err := walk(id); err != nil {
				return err
			}
		}
	}
	return nil
}

// Ready returns the tasks that are ready to be worked, in id order. It fails if the
// graph has a cycle rather than producing a partial or looping answer.
func (s *TaskSet) Ready() ([]*Task, error) {
	if err := s.DetectCycle(); err != nil {
		return nil, err
	}
	var out []*Task
	for _, id := range s.sortedIDs() {
		t := s.Tasks[id]
		if t.Status == StatusTodo && len(s.Blockers(id)) == 0 {
			out = append(out, t)
		}
	}
	return out, nil
}

// Next returns the highest-priority ready task, or nil if none. Priority is id
// order, which follows creation order; ties cannot occur since ids are unique.
func (s *TaskSet) Next() (*Task, error) {
	ready, err := s.Ready()
	if err != nil {
		return nil, err
	}
	if len(ready) == 0 {
		return nil, nil
	}
	return ready[0], nil
}
