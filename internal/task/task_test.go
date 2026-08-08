package task

import (
	"bytes"
	"strings"
	"testing"

	"github.com/juststeveking/flow/internal/core"
)

// sample is the prompt's exemplar task: nested criteria and a code fence that
// itself contains `- [ ]`, which must never be mistaken for a criterion.
const sample = `---
id: T-014
title: Rate limit the ingestion endpoint
status: doing
assignee: agent:claude
depends: [T-011]
tags: [api, ingestion]
created: 2026-08-01
updated: 2026-08-07
---
## Context
Ingestion accepts unbounded writes from any authenticated key.

## Acceptance criteria
- [x] Per key token bucket, 100 req/min
- [ ] 429 responses use Problem+JSON
  - [ ] error body carries a type URI
- [ ] Feature test covers burst
  then recovery window

Example, not a criterion:

` + "```" + `
- [ ] this lives in a code fence
` + "```" + `

## Log
- 2026-08-07 claude: middleware added, burst test failing on window reset
`

func mustParse(t *testing.T, src string) *Task {
	t.Helper()
	tk, derrs, transient := Parse("mem://task", []byte(src))
	if transient != nil {
		t.Fatalf("unexpected transient error: %v", transient)
	}
	if fatal(derrs) {
		t.Fatalf("unexpected fatal parse errors: %v", derrs)
	}
	return tk
}

func fatal(errs []core.DocError) bool {
	for _, e := range errs {
		if e.Severity == core.SevError {
			return true
		}
	}
	return false
}

// TestBytesRoundTripNoop is the guard rail: parse then write with no mutation must
// be byte identical, across CRLF, unknown fields, and no trailing newline.
func TestBytesRoundTripNoop(t *testing.T) {
	cases := map[string]string{
		"canonical":       sample,
		"unknown fields":  "---\nid: T-001\ntitle: X\nstatus: todo\npriority: high\nowner: nobody\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\nBody with a trailing newline.\n",
		"no trailing nl":  "---\nid: T-002\ntitle: Y\nstatus: todo\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\nNo final newline here.",
		"crlf":            "---\r\nid: T-003\r\ntitle: Z\r\nstatus: todo\r\ncreated: 2026-08-01\r\nupdated: 2026-08-01\r\n---\r\nWindows body.\r\n",
		"empty body":      "---\nid: T-004\ntitle: W\nstatus: todo\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n",
		"flow list style": "---\nid: T-005\ntitle: V\nstatus: todo\ndepends: [T-001, T-002]\ntags: [a, b]\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\nBody.\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			tk, _, transient := Parse("mem://task", []byte(src))
			if transient != nil {
				t.Fatalf("transient: %v", transient)
			}
			if got := string(tk.Bytes()); got != src {
				t.Fatalf("no-op write not byte identical\n--- want ---\n%q\n--- got ---\n%q", src, got)
			}
		})
	}
}

// FuzzParseWriteRoundTrip asserts the no-op identity for arbitrary inputs: whenever
// the frontmatter framing parses, Bytes must reproduce the input exactly.
func FuzzParseWriteRoundTrip(f *testing.F) {
	f.Add(sample)
	f.Add("not a flow doc at all")
	f.Add("---\nid: T-1\n---\nbody")
	f.Fuzz(func(t *testing.T, src string) {
		tk, _, transient := Parse("mem://fuzz", []byte(src))
		if transient != nil || tk == nil {
			return // no intact frontmatter framing; nothing to round-trip
		}
		if got := string(tk.Bytes()); got != src {
			t.Fatalf("round-trip mismatch\nin:  %q\nout: %q", src, got)
		}
	})
}

func TestCriteriaSkipsCodeFenceAndCountsNested(t *testing.T) {
	tk := mustParse(t, sample)
	done, total, err := tk.Progress()
	if err != nil {
		t.Fatal(err)
	}
	// Four real criteria (one ticked); the code-fence line is not one of them.
	if total != 4 {
		t.Fatalf("total criteria = %d, want 4 (code fence must be excluded)", total)
	}
	if done != 1 {
		t.Fatalf("done criteria = %d, want 1", done)
	}
}

func TestCheckPatchesCorrectBoxOnly(t *testing.T) {
	tk := mustParse(t, sample)
	before := append([]byte(nil), tk.Bytes()...)

	if err := tk.Check(2); err != nil { // tick "429 responses use Problem+JSON"
		t.Fatal(err)
	}
	after := tk.Bytes()

	if len(before) != len(after) {
		t.Fatalf("check changed the length: %d -> %d (must be a single-byte patch)", len(before), len(after))
	}
	diff := 0
	pos := -1
	for i := range before {
		if before[i] != after[i] {
			diff++
			pos = i
		}
	}
	if diff != 1 {
		t.Fatalf("expected exactly one changed byte, got %d", diff)
	}
	// The patched byte must be the state char of the second criterion and now 'x'.
	if after[pos] != 'x' || before[pos] != ' ' {
		t.Fatalf("patched byte = %q (was %q), want ' ' -> 'x'", after[pos], before[pos])
	}
	if !strings.Contains(string(after), "- [x] 429 responses use Problem+JSON") {
		t.Fatalf("second criterion not ticked:\n%s", after)
	}
	// The code-fence line must remain untouched.
	if !strings.Contains(string(after), "- [ ] this lives in a code fence") {
		t.Fatal("code-fence checkbox was altered")
	}

	done, _, _ := tk.Progress()
	if done != 2 {
		t.Fatalf("done after check = %d, want 2", done)
	}
}

func TestCheckIsIdempotentByteIdentical(t *testing.T) {
	tk := mustParse(t, sample)
	before := append([]byte(nil), tk.Bytes()...)
	if err := tk.Check(1); err != nil { // already ticked
		t.Fatal(err)
	}
	if !bytes.Equal(before, tk.Bytes()) {
		t.Fatal("re-checking an already-ticked box must be a no-op")
	}
}

func TestCheckOutOfRange(t *testing.T) {
	tk := mustParse(t, sample)
	if err := tk.Check(99); err == nil {
		t.Fatal("expected an out-of-range error")
	}
}

func TestAppendLogToExistingSection(t *testing.T) {
	tk := mustParse(t, sample)
	body := string(tk.Body())
	// Everything before ## Log must be preserved verbatim.
	prefix := body[:strings.Index(body, "## Log")]

	if err := tk.AppendLog("2026-08-08 claude: window reset fixed"); err != nil {
		t.Fatal(err)
	}
	got := string(tk.Body())
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("body before Log changed:\n%s", got)
	}
	if !strings.Contains(got, "- 2026-08-08 claude: window reset fixed") {
		t.Fatalf("new log line missing:\n%s", got)
	}
	// The pre-existing entry must survive.
	if !strings.Contains(got, "- 2026-08-07 claude: middleware added") {
		t.Fatalf("existing log line lost:\n%s", got)
	}
	if _, derrs, transient := Parse(tk.Path, tk.Bytes()); transient != nil || fatal(derrs) {
		t.Fatalf("document no longer parses after log append: %v %v", transient, derrs)
	}
}

func TestAppendLogCreatesSection(t *testing.T) {
	src := "---\nid: T-020\ntitle: No log yet\nstatus: todo\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n## Context\nSomething.\n"
	tk := mustParse(t, src)
	if err := tk.AppendLog("2026-08-08 human:steve: created log"); err != nil {
		t.Fatal(err)
	}
	got := string(tk.Body())
	if !strings.Contains(got, "## Log") {
		t.Fatalf("Log section not created:\n%s", got)
	}
	if !strings.Contains(got, "- 2026-08-08 human:steve: created log") {
		t.Fatalf("log entry missing:\n%s", got)
	}
	if !strings.HasPrefix(got, "## Context\nSomething.\n") {
		t.Fatalf("original body not preserved:\n%s", got)
	}
	if _, derrs, transient := Parse(tk.Path, tk.Bytes()); transient != nil || fatal(derrs) {
		t.Fatalf("document no longer parses: %v %v", transient, derrs)
	}
}

func TestSetStatusPreservesBodyAndUnknownFields(t *testing.T) {
	src := "---\nid: T-030\ntitle: Keep me\nstatus: todo\npriority: high\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n## Context\nProse the renderer must not touch.\n"
	tk := mustParse(t, src)
	body := string(tk.Body())

	if err := tk.SetStatus(StatusDoing); err != nil {
		t.Fatal(err)
	}
	if err := tk.Touch("2026-08-08"); err != nil {
		t.Fatal(err)
	}

	reparsed, derrs, transient := Parse(tk.Path, tk.Bytes())
	if transient != nil || fatal(derrs) {
		t.Fatalf("re-parse failed: %v %v", transient, derrs)
	}
	if reparsed.Status != StatusDoing {
		t.Fatalf("status = %q, want doing", reparsed.Status)
	}
	if reparsed.Updated != "2026-08-08" {
		t.Fatalf("updated = %q, want 2026-08-08", reparsed.Updated)
	}
	if reparsed.Extra["priority"] != "high" {
		t.Fatalf("unknown field priority not preserved: %#v", reparsed.Extra)
	}
	if string(reparsed.Body()) != body {
		t.Fatalf("body was reformatted:\n--- want ---\n%q\n--- got ---\n%q", body, reparsed.Body())
	}
}

func TestSetStatusRejectsIllegal(t *testing.T) {
	tk := mustParse(t, sample)
	if err := tk.SetStatus(Status("nonsense")); err == nil {
		t.Fatal("expected illegal-status error")
	}
}

func TestParseValidation(t *testing.T) {
	cases := map[string]string{
		"bad status":    "---\nid: T-1\ntitle: X\nstatus: wat\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n",
		"bad id":        "---\nid: nope\ntitle: X\nstatus: todo\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n",
		"bad assignee":  "---\nid: T-1\ntitle: X\nstatus: todo\nassignee: claude\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n",
		"missing title": "---\nid: T-1\nstatus: todo\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n",
		"bad depends":   "---\nid: T-1\ntitle: X\nstatus: todo\ndepends: [nope]\ncreated: 2026-08-01\nupdated: 2026-08-01\n---\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			_, derrs, transient := Parse("mem://task", []byte(src))
			if transient != nil {
				t.Fatalf("unexpected transient: %v", transient)
			}
			if !fatal(derrs) {
				t.Fatalf("expected a fatal validation error, got %v", derrs)
			}
		})
	}
}

func TestUnterminatedFrontmatterIsTransient(t *testing.T) {
	src := "---\nid: T-1\ntitle: half written\n"
	_, _, transient := Parse("mem://task", []byte(src))
	if transient != core.ErrUnterminatedFrontmatter {
		t.Fatalf("transient = %v, want ErrUnterminatedFrontmatter", transient)
	}
}

func TestNoFrontmatterIsFatalNotTransient(t *testing.T) {
	_, derrs, transient := Parse("mem://task", []byte("just some text\n"))
	if transient != nil {
		t.Fatalf("transient = %v, want nil", transient)
	}
	if !fatal(derrs) {
		t.Fatal("expected a fatal error for a document with no frontmatter")
	}
}
