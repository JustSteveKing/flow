package task

import "testing"

func TestDoneNotReachableFromDoing(t *testing.T) {
	tk := &Task{ID: "T-001", Status: StatusDoing}
	if err := TransitionTask(tk, StatusDone); err == nil {
		t.Fatal("doing -> done must be illegal (path is doing -> review -> done)")
	}
	if tk.Status != StatusDoing {
		t.Fatalf("status changed on a rejected transition: %s", tk.Status)
	}
}

func TestReviewThenDonePath(t *testing.T) {
	tk := mustParse(t, "---\nid: T-002\ntitle: X\nstatus: doing\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n")
	if err := TransitionTask(tk, StatusReview); err != nil {
		t.Fatalf("doing -> review should be legal: %v", err)
	}
	if err := TransitionTask(tk, StatusDone); err != nil {
		t.Fatalf("review -> done should be legal: %v", err)
	}
	// The frontmatter must reflect the final status after re-serialisation.
	reparsed := mustParse(t, string(tk.Bytes()))
	if reparsed.Status != StatusDone {
		t.Fatalf("persisted status = %s, want done", reparsed.Status)
	}
}

func TestReviewCompleteGuard(t *testing.T) {
	partial := mustParse(t, sample) // one of four criteria ticked
	ok, err := ReviewComplete(partial)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("review should be blocked while criteria are unticked")
	}

	none := mustParse(t, "---\nid: T-003\ntitle: X\nstatus: doing\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n## Acceptance criteria\n")
	ok, err = ReviewComplete(none)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("a task with no criteria may enter review")
	}
}

func TestSameStatusIsNoop(t *testing.T) {
	tk := mustParse(t, sample) // status doing
	before := string(tk.Bytes())
	if err := TransitionTask(tk, StatusDoing); err != nil {
		t.Fatal(err)
	}
	if string(tk.Bytes()) != before {
		t.Fatal("no-op transition must not rewrite the file")
	}
}
