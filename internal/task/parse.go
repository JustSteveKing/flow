package task

import (
	"bytes"
	"fmt"

	"github.com/juststeveking/flow/internal/core"
	"gopkg.in/yaml.v3"
)

// errbag accumulates DocErrors for one task during validation. It mirrors core's
// unexported errbag; the package boundary makes a small parallel worth the little
// duplication over exporting internals.
type errbag struct {
	path string
	errs []core.DocError
}

func (b *errbag) errf(field, format string, a ...any) {
	b.errs = append(b.errs, core.DocError{Path: b.path, Field: field, Severity: core.SevError, Message: fmt.Sprintf(format, a...)})
}

func req(b *errbag, field, val string) {
	if val == "" {
		b.errf(field, "required")
	}
}

// Parse parses one task document. It returns the Task (non-nil whenever the
// frontmatter framing is intact, so callers can inspect it), any validation
// DocErrors, and a transient error for the half-written signature.
//
// A transient ErrUnterminatedFrontmatter (reused from core) is returned via the
// third value so a caller can retry after the debounce window rather than treat a
// file caught mid-save as invalid.
func Parse(path string, src []byte) (*Task, []core.DocError, error) {
	fmStart, fmEnd, bodyStart, err := frontmatterSpan(src)
	if err != nil {
		if err == core.ErrUnterminatedFrontmatter {
			return nil, nil, err
		}
		return nil, []core.DocError{{Path: path, Severity: core.SevError, Message: err.Error()}}, nil
	}

	var t Task
	if err := yaml.Unmarshal(src[fmStart:fmEnd], &t); err != nil {
		return nil, []core.DocError{{Path: path, Severity: core.SevError, Message: "invalid YAML frontmatter: " + err.Error()}}, nil
	}
	t.Path = path
	t.src = append([]byte(nil), src...) // own copy: mutations splice into src in place
	t.fmStart, t.fmEnd, t.bodyStart = fmStart, fmEnd, bodyStart

	return &t, t.validate(), nil
}

// frontmatterSpan locates the leading `---`-fenced YAML block and returns byte
// offsets into src: the inner YAML is src[fmStart:fmEnd] (fmEnd is the start of
// the closing fence line, so the inner block keeps its own trailing newline), and
// the body begins at bodyStart. It retains offsets rather than reconstructing the
// text, which is what lets writes splice surgically. Fence rules match core's
// splitFrontmatter: the first line and a later line must each be exactly "---".
func frontmatterSpan(src []byte) (fmStart, fmEnd, bodyStart int, err error) {
	nl := bytes.IndexByte(src, '\n')
	if nl < 0 || !isFence(src[:nl]) {
		return 0, 0, 0, core.ErrNoFrontmatter
	}
	fmStart = nl + 1
	for pos := fmStart; pos <= len(src); {
		rel := bytes.IndexByte(src[pos:], '\n')
		lineEnd := len(src)
		if rel >= 0 {
			lineEnd = pos + rel
		}
		if isFence(src[pos:lineEnd]) {
			if rel < 0 {
				return fmStart, pos, len(src), nil
			}
			return fmStart, pos, lineEnd + 1, nil
		}
		if rel < 0 {
			break // EOF with no closing fence
		}
		pos = lineEnd + 1
	}
	return 0, 0, 0, core.ErrUnterminatedFrontmatter
}

// isFence reports whether a line (CR tolerated) is exactly "---".
func isFence(line []byte) bool {
	return string(bytes.TrimRight(line, "\r")) == "---"
}
