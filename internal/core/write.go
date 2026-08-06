package core

import (
	"bytes"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Render serialises a flow document back to markdown: the YAML frontmatter block
// followed by the preserved markdown body. Field order follows the struct
// declaration order, which matches SCHEMA.md, so writes are deterministic.
//
// Manifest is pure YAML with no body; pass a *Manifest to get bare YAML.
func Render(doc any) ([]byte, error) {
	if m, ok := doc.(*Manifest); ok {
		return yaml.Marshal(m)
	}
	body := bodyOf(doc)
	fm, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fm)
	buf.WriteString("---\n")
	if body != "" {
		buf.WriteString("\n")
		buf.WriteString(body)
		if body[len(body)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes(), nil
}

func bodyOf(doc any) string {
	switch v := doc.(type) {
	case *Pitch:
		return v.Body
	case *Spec:
		return v.Body
	case *Decision:
		return v.Body
	case *Cycle:
		return v.Body
	}
	return ""
}

// RelPath returns the `.flow/`-relative path a document belongs at, derived from
// its identity. This is the single place document identity maps to a location.
func RelPath(doc any) (string, error) {
	switch v := doc.(type) {
	case *Pitch:
		return filepath.Join(".flow", "pitches", v.Lineage+".md"), nil
	case *Spec:
		return filepath.Join(".flow", "specs", v.Lineage+".md"), nil
	case *Cycle:
		return filepath.Join(".flow", "cycles", v.ID+".md"), nil
	case *Decision:
		return filepath.Join(".flow", "decisions", string(v.Layer), v.ID+".md"), nil
	case *Manifest:
		return filepath.Join(".flow", "manifest.yaml"), nil
	default:
		return "", fmt.Errorf("cannot route unknown document type %T", doc)
	}
}
