package core

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
)

// ErrNoFrontmatter means the file did not begin with a `---` fence.
var ErrNoFrontmatter = errors.New("missing opening frontmatter fence")

// ErrUnterminatedFrontmatter means the opening fence had no closing `---`.
// This is the signature of a half-written file caught mid-save; callers should
// treat it as transient and retry after the debounce window rather than drop.
var ErrUnterminatedFrontmatter = errors.New("unterminated frontmatter block")

// SplitFrontmatter separates the leading YAML frontmatter bytes from the markdown
// body. Exported for presentation layers that render the two parts differently.
func SplitFrontmatter(content []byte) (frontmatter []byte, body string, err error) {
	return splitFrontmatter(content)
}

// splitFrontmatter separates the leading YAML frontmatter from the markdown body.
// A document MUST start with a line that is exactly "---" and the block MUST be
// closed by a later line that is exactly "---". Returns the frontmatter bytes
// (without fences) and the body.
func splitFrontmatter(content []byte) (frontmatter []byte, body string, err error) {
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	if !sc.Scan() {
		return nil, "", ErrNoFrontmatter
	}
	if strings.TrimRight(sc.Text(), "\r") != "---" {
		return nil, "", ErrNoFrontmatter
	}

	var fm strings.Builder
	closed := false
	var bodyLines []string
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if !closed {
			if line == "---" {
				closed = true
				continue
			}
			fm.WriteString(sc.Text())
			fm.WriteByte('\n')
			continue
		}
		bodyLines = append(bodyLines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, "", err
	}
	if !closed {
		return nil, "", ErrUnterminatedFrontmatter
	}
	return []byte(fm.String()), strings.TrimLeft(strings.Join(bodyLines, "\n"), "\n"), nil
}
