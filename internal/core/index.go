package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectIndex is a rebuildable projection of one project's `.flow/` tree.
// It is never authoritative; the files are (adr-0001). Reload to refresh.
type ProjectIndex struct {
	Root      string // project root (the directory containing .flow)
	Manifest  *Manifest
	Pitches   map[string]*Pitch    // by lineage
	Specs     map[string]*Spec     // by lineage
	Cycles    map[string]*Cycle    // by cycle id
	Decisions map[string]*Decision // by adr id, local shadows baseline
	AllADRs   []*Decision          // every ADR incl. shadowed baseline, for the graph
	Errors    []DocError

	// Transient holds paths that failed with a half-written signature and should
	// be retried after the debounce window rather than treated as invalid.
	Transient []string
}

// FlowDir returns the .flow directory for a project root.
func FlowDir(root string) string { return filepath.Join(root, ".flow") }

// LoadProject indexes the `.flow/` tree under root. Missing or invalid documents
// are recorded in Errors; fatal documents are dropped from the maps. A missing
// manifest is a fatal, index-level error.
func LoadProject(root string) (*ProjectIndex, error) {
	idx := &ProjectIndex{
		Root:      root,
		Pitches:   map[string]*Pitch{},
		Specs:     map[string]*Spec{},
		Cycles:    map[string]*Cycle{},
		Decisions: map[string]*Decision{},
	}
	flow := FlowDir(root)

	// manifest first
	mpath := filepath.Join(flow, "manifest.yaml")
	mb, err := os.ReadFile(mpath)
	if err != nil {
		return nil, err
	}
	m, merrs := ParseManifest(mpath, mb)
	idx.Errors = append(idx.Errors, merrs...)
	idx.Manifest = m

	idx.loadDir(filepath.Join(flow, "pitches"))
	idx.loadDir(filepath.Join(flow, "specs"))
	idx.loadDir(filepath.Join(flow, "cycles"))
	idx.loadDir(filepath.Join(flow, "decisions", "baseline"))
	idx.loadDir(filepath.Join(flow, "decisions", "local"))

	idx.resolveReferences()
	return idx, nil
}

// ProjectID returns the manifest project id, or "" if unindexed.
func (idx *ProjectIndex) ProjectID() string {
	if idx.Manifest == nil {
		return ""
	}
	return idx.Manifest.ProjectID
}

func (idx *ProjectIndex) loadDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return // absent optional directory is fine
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || ignoredName(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			idx.Errors = append(idx.Errors, DocError{Path: path, Severity: SevError, Message: err.Error()})
			continue
		}
		idx.ingest(path, content)
	}
}

// ingest parses one document's bytes and files it into the index.
func (idx *ProjectIndex) ingest(path string, content []byte) {
	doc, derrs, transient := ParseDocument(path, content)
	if transient != nil {
		idx.Transient = append(idx.Transient, path)
		return
	}
	idx.Errors = append(idx.Errors, derrs...)
	if fatal(derrs) {
		return // dropped from the index (adr-level MUST violation)
	}
	switch v := doc.(type) {
	case *Pitch:
		idx.Pitches[v.Lineage] = v
	case *Spec:
		idx.Specs[v.Lineage] = v
	case *Cycle:
		idx.Cycles[v.ID] = v
	case *Decision:
		idx.AllADRs = append(idx.AllADRs, v)
		// local shadows baseline on id clash
		if existing, ok := idx.Decisions[v.ID]; !ok || existing.Layer == LayerBaseline {
			idx.Decisions[v.ID] = v
		}
	}
}

func fatal(errs []DocError) bool {
	for _, e := range errs {
		if e.fatal() {
			return true
		}
	}
	return false
}

// ignoredName filters editor scratch and dotfiles.
func ignoredName(name string) bool {
	if strings.HasPrefix(name, ".") || strings.HasSuffix(name, "~") {
		return true
	}
	if strings.HasSuffix(name, ".swp") || strings.HasSuffix(name, ".tmp") {
		return true
	}
	return false
}

// resolveReferences applies the cross-document rules from SCHEMA.md. Problems are
// recorded as index-level errors; documents already indexed are not removed.
func (idx *ProjectIndex) resolveReferences() {
	resolveADR := func(id string) bool { _, ok := idx.Decisions[id]; return ok }

	// pitch assumes_decisions resolve
	for _, p := range idx.Pitches {
		for _, id := range p.AssumesDecisions {
			if !resolveADR(id) {
				idx.Errors = append(idx.Errors, DocError{Path: p.Path, Field: "assumes_decisions", Severity: SevError, Message: "unresolved decision " + id})
			}
		}
	}

	// spec references
	for _, s := range idx.Specs {
		if _, ok := idx.Pitches[s.Lineage]; !ok {
			idx.Errors = append(idx.Errors, DocError{Path: s.Path, Field: "lineage", Severity: SevError, Message: "no pitch for lineage " + s.Lineage})
		}
		if _, ok := idx.Cycles[s.Cycle]; !ok {
			idx.Errors = append(idx.Errors, DocError{Path: s.Path, Field: "cycle", Severity: SevError, Message: "no cycle " + s.Cycle})
		}
		for _, id := range s.AssumesDecisions {
			if !resolveADR(id) {
				idx.Errors = append(idx.Errors, DocError{Path: s.Path, Field: "assumes_decisions", Severity: SevError, Message: "unresolved decision " + id})
			}
		}
	}

	// cycle bets resolve to a pitch whose status is bet
	for _, c := range idx.Cycles {
		for _, bet := range c.Bets {
			p, ok := idx.Pitches[bet.Lineage]
			if !ok {
				idx.Errors = append(idx.Errors, DocError{Path: c.Path, Field: "bets", Severity: SevError, Message: "bet on unknown pitch " + bet.Lineage})
				continue
			}
			if p.Status != PitchBet {
				idx.Errors = append(idx.Errors, DocError{Path: c.Path, Field: "bets", Severity: SevWarning, Message: "pitch " + bet.Lineage + " is bet in cycle but its status is " + string(p.Status)})
			}
		}
	}

	idx.checkSupersession()
}

// checkSupersession verifies the ADR chain is acyclic and mutually consistent.
func (idx *ProjectIndex) checkSupersession() {
	// forward-edge consistency: A.superseded_by=B implies B.supersedes=A
	byID := map[string]*Decision{}
	for _, d := range idx.AllADRs {
		byID[d.ID] = d
	}
	for _, d := range idx.AllADRs {
		if d.SupersededBy != nil && *d.SupersededBy != "" {
			tgt, ok := byID[*d.SupersededBy]
			if !ok {
				idx.Errors = append(idx.Errors, DocError{Path: d.Path, Field: "superseded_by", Severity: SevError, Message: "points at unknown adr " + *d.SupersededBy})
				continue
			}
			if tgt.Supersedes == nil || *tgt.Supersedes != d.ID {
				idx.Errors = append(idx.Errors, DocError{Path: d.Path, Field: "superseded_by", Severity: SevError, Message: "chain inconsistent: " + tgt.ID + " does not supersede " + d.ID})
			}
			// baseline must not reference a local record (Decision 4)
			if d.Layer == LayerBaseline && tgt.Layer == LayerLocal {
				idx.Errors = append(idx.Errors, DocError{Path: d.Path, Field: "superseded_by", Severity: SevError, Message: "baseline record superseded by local record"})
			}
		}
	}

	// cycle detection over superseded_by edges
	const (
		white = 0
		grey  = 1
		black = 2
	)
	colour := map[string]int{}
	var walk func(id string) bool // returns true if a cycle is found
	walk = func(id string) bool {
		colour[id] = grey
		d := byID[id]
		if d != nil && d.SupersededBy != nil && *d.SupersededBy != "" {
			nxt := *d.SupersededBy
			switch colour[nxt] {
			case grey:
				return true
			case white:
				if walk(nxt) {
					return true
				}
			}
		}
		colour[id] = black
		return false
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids) // deterministic
	for _, id := range ids {
		if colour[id] == white {
			if walk(id) {
				idx.Errors = append(idx.Errors, DocError{Path: idx.Root, Field: "supersession", Severity: SevError, Message: "cycle in supersession chain involving " + id})
				break
			}
		}
	}
}

// ADRHeads returns the current (non-superseded) head of every supersession chain.
func (idx *ProjectIndex) ADRHeads() []*Decision {
	var heads []*Decision
	for _, d := range idx.Decisions {
		if d.Status != DecisionSuperseded {
			heads = append(heads, d)
		}
	}
	sort.Slice(heads, func(i, j int) bool { return heads[i].ID < heads[j].ID })
	return heads
}
