package task

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// This file holds the surgical mutations. Two invariants govern every one of them:
// the human-authored body is never round-tripped through goldmark's renderer, and
// applying no mutation leaves the bytes identical. Frontmatter changes re-serialise
// only the YAML block between the fences; body changes patch bytes in place.

// SetStatus sets the frontmatter status and re-serialises the frontmatter block.
// Callers bump Updated (via Touch) as part of any mutating write.
func (t *Task) SetStatus(s Status) error {
	if !ValidStatus(s) {
		return fmt.Errorf("illegal status %q", s)
	}
	t.Status = s
	return t.reserialiseFrontmatter()
}

// SetAssignee sets assignee to agent:{name} or human:{name}.
func (t *Task) SetAssignee(a string) error {
	if a != "" && !ValidAssignee(a) {
		return fmt.Errorf("malformed assignee %q (want agent:name or human:name)", a)
	}
	t.Assignee = a
	return t.reserialiseFrontmatter()
}

// ClearAssignee drops the assignee (omitempty removes the line).
func (t *Task) ClearAssignee() error {
	t.Assignee = ""
	return t.reserialiseFrontmatter()
}

// Touch sets updated to date and re-serialises the frontmatter. Per the subsystem
// contract, every mutating write bumps updated; callers apply this after a change.
func (t *Task) Touch(date string) error {
	t.Updated = date
	return t.reserialiseFrontmatter()
}

// reserialiseFrontmatter re-marshals the struct and splices it between the fences,
// leaving the opening fence, closing fence, and body bytes exactly as they were.
// Unknown fields survive via the inline Extra map. Field order follows the struct
// declaration, so the write is deterministic.
func (t *Task) reserialiseFrontmatter() error {
	fm, err := yaml.Marshal(t)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.Grow(t.fmStart + len(fm) + len(t.src) - t.fmEnd)
	buf.Write(t.src[:t.fmStart])
	buf.Write(fm)
	buf.Write(t.src[t.fmEnd:])
	delta := len(fm) - (t.fmEnd - t.fmStart)
	t.src = buf.Bytes()
	t.fmEnd += delta
	t.bodyStart += delta
	return nil
}

// Check ticks the acceptance criterion at the given 1-based index.
func (t *Task) Check(oneBased int) error { return t.setCheckbox(oneBased, true) }

// Uncheck unticks the acceptance criterion at the given 1-based index.
func (t *Task) Uncheck(oneBased int) error { return t.setCheckbox(oneBased, false) }

// setCheckbox flips exactly the two bytes at `[ ]` / `[x]` of one criterion in
// place, touching nothing else. Already being in the desired state is a no-op, so
// the bytes stay identical.
func (t *Task) setCheckbox(oneBased int, checked bool) error {
	cs, err := t.criteria()
	if err != nil {
		return err
	}
	if oneBased < 1 || oneBased > len(cs) {
		return fmt.Errorf("no acceptance criterion %d (have %d)", oneBased, len(cs))
	}
	c := cs[oneBased-1]
	if c.checked == checked {
		return nil
	}
	if checked {
		t.src[c.statePos] = 'x'
	} else {
		t.src[c.statePos] = ' '
	}
	return nil
}

// Progress counts ticked and total acceptance criteria, the fraction shown on the
// board and in the generated index.
func (t *Task) Progress() (done, total int, err error) {
	cs, err := t.criteria()
	if err != nil {
		return 0, 0, err
	}
	for _, c := range cs {
		if c.checked {
			done++
		}
	}
	return done, len(cs), nil
}

// AppendLog appends a dated entry to the Log section as a new list item `- entry`,
// creating the section at the end of the body if it is missing. The caller supplies
// the entry text (for example "2026-08-07 claude: message"); this splices it in
// without reformatting the surrounding prose.
func (t *Task) AppendLog(entry string) error {
	entry = strings.TrimRight(entry, "\r\n")
	body := t.Body()
	doc := parse(body)
	hs, he := sectionBounds(doc, body, "log")

	if hs < 0 {
		var b bytes.Buffer
		b.Write(body)
		if len(body) > 0 {
			if body[len(body)-1] != '\n' {
				b.WriteByte('\n')
			}
			b.WriteByte('\n') // blank line before the new section
		}
		b.WriteString("## Log\n\n- ")
		b.WriteString(entry)
		b.WriteByte('\n')
		t.replaceBody(b.Bytes())
		return nil
	}

	// Insert after the last non-blank byte of the existing section.
	end := he
	for end > hs && isSpaceOrNL(body[end-1]) {
		end--
	}
	var b bytes.Buffer
	b.Write(body[:end])
	b.WriteString("\n- ")
	b.WriteString(entry)
	b.Write(body[end:])
	t.replaceBody(b.Bytes())
	return nil
}

// replaceBody swaps the body bytes, leaving the frontmatter span untouched.
func (t *Task) replaceBody(newBody []byte) {
	var b bytes.Buffer
	b.Grow(t.bodyStart + len(newBody))
	b.Write(t.src[:t.bodyStart])
	b.Write(newBody)
	t.src = b.Bytes()
}

// criterion locates one acceptance-criteria checkbox: statePos is the absolute
// index into src of the byte between the brackets.
type criterion struct {
	statePos int
	checked  bool
}

// criteria returns the acceptance-criteria checkboxes in document order. It uses
// goldmark's GFM task-list parsing so that `- [ ]` inside a code fence, or lists
// under other headings, are never mistaken for criteria. Offsets from the parsed
// body are shifted by bodyStart to index into src.
func (t *Task) criteria() ([]criterion, error) {
	body := t.Body()
	doc := parse(body)
	hs, he := sectionBounds(doc, body, "acceptance criteria")
	if hs < 0 {
		return nil, nil
	}
	var out []criterion
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		cb, ok := n.(*east.TaskCheckBox)
		if !ok {
			return ast.WalkContinue, nil
		}
		start := blockLineStart(n)
		if start < hs || start >= he {
			return ast.WalkContinue, nil
		}
		bracket := start
		if bracket >= len(body) || body[bracket] != '[' {
			// Defend against layouts where the item text does not begin flush at
			// the bracket by scanning forward to the marker.
			j := bytes.IndexByte(body[start:], '[')
			if j < 0 {
				return ast.WalkContinue, nil
			}
			bracket = start + j
		}
		out = append(out, criterion{statePos: t.bodyStart + bracket + 1, checked: cb.IsChecked})
		return ast.WalkContinue, nil
	})
	return out, err
}

// parse builds a fresh GFM parse of body. A new parser per call avoids sharing
// mutable parser state; task documents are small and this is not a hot path.
func parse(body []byte) ast.Node {
	return goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(text.NewReader(body))
}

type hdr struct {
	level int
	start int
	text  string
}

// sectionBounds returns the body-relative [start, end) byte range of the section
// under the heading whose text (case-insensitively, trimmed) equals name. start is
// the heading's text offset; end is the next heading of equal or shallower level,
// or the end of the body. It returns (-1, -1) when no such heading exists.
func sectionBounds(doc ast.Node, body []byte, name string) (start, end int) {
	hs := headings(doc, body)
	for i, h := range hs {
		if h.text != name {
			continue
		}
		end = len(body)
		for _, h2 := range hs[i+1:] {
			if h2.level <= h.level {
				end = h2.start
				break
			}
		}
		return h.start, end
	}
	return -1, -1
}

// headings collects every heading with its level, text offset, and lowercased text.
func headings(doc ast.Node, body []byte) []hdr {
	var out []hdr
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		start, txt := -1, ""
		if ls := h.Lines(); ls != nil && ls.Len() > 0 {
			seg := ls.At(0)
			start = seg.Start
			txt = strings.ToLower(strings.TrimSpace(string(seg.Value(body))))
		}
		out = append(out, hdr{level: h.Level, start: start, text: txt})
		return ast.WalkContinue, nil
	})
	return out
}

// blockLineStart climbs to the nearest enclosing block with source lines and
// returns the start offset of its first line, where a task item's `[` sits.
func blockLineStart(n ast.Node) int {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Type() == ast.TypeBlock {
			if ls := p.Lines(); ls != nil && ls.Len() > 0 {
				return ls.At(0).Start
			}
		}
	}
	return -1
}

func isSpaceOrNL(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
