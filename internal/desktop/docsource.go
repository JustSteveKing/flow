package desktop

import (
	"net/http"
	"os"
	"strings"

	"github.com/juststeveking/flow/internal/core"
	"gopkg.in/yaml.v3"
)

// DocRaw returns the exact on-disk markdown of one indexed document, resolved by
// project, kind, and id. It reads only paths already in the index, so there is no
// path traversal: you can view what the index sees, which is the file itself
// (adr-0001, files are truth).
func (w *Workspace) DocRaw(projectID, kind, id string) (path, content string, ok bool) {
	idx := w.snapshot()
	if idx == nil {
		return "", "", false
	}
	p, exists := idx.Projects[projectID]
	if !exists {
		return "", "", false
	}
	switch kind {
	case "pitch":
		if d := p.Pitches[id]; d != nil {
			path = d.Path
		}
	case "spec":
		if d := p.Specs[id]; d != nil {
			path = d.Path
		}
	case "cycle":
		if d := p.Cycles[id]; d != nil {
			path = d.Path
		}
	case "decision":
		if d := p.Decisions[id]; d != nil {
			path = d.Path
		}
	case "manifest":
		if p.Manifest != nil {
			path = p.Manifest.Path
		}
	}
	if path == "" {
		return "", "", false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}
	return path, string(b), true
}

// Field is one frontmatter key rendered as a table row. Value is a display
// string: scalars verbatim, lists and maps as readable YAML.
type Field struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DocView is the payload for the source viewer: the raw content, plus the
// frontmatter split into fields and the markdown body, for richer rendering.
type DocView struct {
	Path        string  `json:"path"`
	Content     string  `json:"content"`
	Frontmatter []Field `json:"frontmatter"`
	Body        string  `json:"body"`
}

func (w *Workspace) handleDoc(rw http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path, content, ok := w.DocRaw(q.Get("project"), q.Get("kind"), q.Get("id"))
	if !ok {
		writeJSON(rw, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}
	view := DocView{Path: path, Content: content}
	if fm, body, err := core.SplitFrontmatter([]byte(content)); err == nil {
		view.Frontmatter = parseFields(fm)
		view.Body = body
	} else {
		// manifest.yaml has no fences; treat the whole file as frontmatter.
		view.Frontmatter = parseFields([]byte(content))
	}
	writeJSON(rw, http.StatusOK, view)
}

// parseFields flattens a top-level YAML mapping into ordered display fields,
// preserving document order via yaml.Node.
func parseFields(fm []byte) []Field {
	var root yaml.Node
	if err := yaml.Unmarshal(fm, &root); err != nil || len(root.Content) == 0 {
		return nil
	}
	m := root.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil
	}
	fields := make([]Field, 0, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		fields = append(fields, Field{Key: m.Content[i].Value, Value: nodeToString(m.Content[i+1])})
	}
	return fields
}

// nodeToString renders a scalar verbatim and any structured node as trimmed YAML.
func nodeToString(n *yaml.Node) string {
	if n.Kind == yaml.ScalarNode {
		return n.Value
	}
	out, err := yaml.Marshal(n)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(out), "\n")
}
