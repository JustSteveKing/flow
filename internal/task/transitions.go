package task

import "fmt"

// The task state machine. The single load-bearing rule is that done is not
// reachable directly from doing: the path is doing -> review -> done, so an agent
// moves finished work to review and stops rather than marking its own work
// complete. The review guard (all criteria ticked) and the blocked reason are
// enforced by the caller, which has the --allow-incomplete flag and the reason text.
var taskTransitions = map[Status]map[Status]bool{
	StatusTodo:    {StatusDoing: true, StatusDropped: true},
	StatusDoing:   {StatusReview: true, StatusBlocked: true, StatusTodo: true, StatusDropped: true},
	StatusBlocked: {StatusDoing: true, StatusTodo: true, StatusDropped: true},
	StatusReview:  {StatusDone: true, StatusDoing: true, StatusDropped: true},
	StatusDone:    {StatusDoing: true}, // reopen for rework
	StatusDropped: {StatusTodo: true},  // revive
}

// IllegalTransitionError reports a rejected task state change.
type IllegalTransitionError struct {
	From, To Status
}

func (e IllegalTransitionError) Error() string {
	return fmt.Sprintf("illegal task transition %s -> %s", e.From, e.To)
}

// TransitionTask moves a task to a new status if the edge is legal, then
// re-serialises the frontmatter. It enforces reachability only; the review and
// blocked guards live with the caller. Callers bump updated via Touch.
func TransitionTask(t *Task, to Status) error {
	if t.Status == to {
		return nil
	}
	if !taskTransitions[t.Status][to] {
		return IllegalTransitionError{From: t.Status, To: to}
	}
	return t.SetStatus(to) // validates the enum and re-serialises
}

// ReviewComplete reports whether a task may enter review without
// --allow-incomplete: every acceptance criterion ticked, or none present.
func ReviewComplete(t *Task) (bool, error) {
	done, total, err := t.Progress()
	if err != nil {
		return false, err
	}
	return total == 0 || done == total, nil
}
