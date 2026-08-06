package core

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// discriminator reads just the `type` field to dispatch parsing.
type discriminator struct {
	Type DocType `yaml:"type"`
}

// ParseDocument parses one markdown flow document (pitch, spec, decision, or
// cycle). It returns the typed value (one of *Pitch, *Spec, *Decision, *Cycle)
// and any DocErrors. If any error is fatal the typed value is still returned for
// inspection, but the caller (the indexer) drops it from the index.
//
// A transient split error (ErrUnterminatedFrontmatter) is returned as a single
// error via the second return so callers can distinguish a half-written file
// from a genuinely invalid one.
func ParseDocument(path string, content []byte) (doc any, derrs []DocError, transient error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		if err == ErrUnterminatedFrontmatter {
			return nil, nil, err
		}
		return nil, []DocError{{Path: path, Severity: SevError, Message: err.Error()}}, nil
	}

	var d discriminator
	if err := yaml.Unmarshal(fm, &d); err != nil {
		return nil, []DocError{{Path: path, Field: "type", Severity: SevError, Message: "invalid YAML frontmatter: " + err.Error()}}, nil
	}

	switch d.Type {
	case TypePitch:
		p, e := parsePitch(path, fm, body)
		return p, e, nil
	case TypeSpec:
		s, e := parseSpec(path, fm, body)
		return s, e, nil
	case TypeDecision:
		dec, e := parseDecision(path, fm, body)
		return dec, e, nil
	case TypeCycle:
		c, e := parseCycle(path, fm, body)
		return c, e, nil
	case "":
		return nil, []DocError{{Path: path, Field: "type", Severity: SevError, Message: "missing type"}}, nil
	default:
		return nil, []DocError{{Path: path, Field: "type", Severity: SevError, Message: fmt.Sprintf("unknown type %q", d.Type)}}, nil
	}
}

func unmarshal(path string, fm []byte, into any) *errbag {
	b := &errbag{path: path}
	if err := yaml.Unmarshal(fm, into); err != nil {
		b.errf("", "invalid YAML frontmatter: %s", err.Error())
	}
	return b
}

func req(b *errbag, field, val string) {
	if val == "" {
		b.errf(field, "required")
	}
}

func parsePitch(path string, fm []byte, body string) (*Pitch, []DocError) {
	var p Pitch
	b := unmarshal(path, fm, &p)
	if b.hasFatal() {
		return &p, b.errs
	}
	p.Path, p.Body = path, body

	req(b, "lineage", p.Lineage)
	if p.Lineage != "" && !ValidLineageID(p.Lineage) {
		b.errf("lineage", "malformed lineage id %q", p.Lineage)
	}
	req(b, "title", p.Title)
	req(b, "created", p.Created)
	req(b, "updated", p.Updated)
	if !pitchStatuses[p.Status] {
		b.errf("status", "illegal status %q", p.Status)
	}
	if !appetites[p.Appetite] {
		b.errf("appetite", "illegal appetite %q", p.Appetite)
	}
	if p.Problem == "" {
		b.warnf("problem", "recommended")
	}
	for _, id := range p.AssumesDecisions {
		if !ValidADRID(id) {
			b.warnf("assumes_decisions", "malformed adr id %q", id)
		}
	}
	return &p, b.errs
}

func parseSpec(path string, fm []byte, body string) (*Spec, []DocError) {
	var s Spec
	b := unmarshal(path, fm, &s)
	if b.hasFatal() {
		return &s, b.errs
	}
	s.Path, s.Body = path, body

	req(b, "lineage", s.Lineage)
	if s.Lineage != "" && !ValidLineageID(s.Lineage) {
		b.errf("lineage", "malformed lineage id %q", s.Lineage)
	}
	req(b, "cycle", s.Cycle)
	if s.Cycle != "" && !ValidCycleID(s.Cycle) {
		b.errf("cycle", "malformed cycle id %q", s.Cycle)
	}
	req(b, "created", s.Created)
	req(b, "updated", s.Updated)
	if !specStatuses[s.Status] {
		b.errf("status", "illegal status %q", s.Status)
	}
	if len(s.Scopes) == 0 {
		b.errf("scopes", "at least one scope required")
	}
	// Interfaces MUST be present once building or frozen.
	if (s.Status == SpecBuilding || s.Status == SpecFrozen) && len(s.Interfaces) == 0 {
		b.errf("interfaces", "interfaces must be locked once status is %q", s.Status)
	}
	ifaceNames := map[string]bool{}
	for i, iface := range s.Interfaces {
		if iface.Name == "" {
			b.errf(fmt.Sprintf("interfaces[%d].name", i), "required")
			continue
		}
		if ifaceNames[iface.Name] {
			b.errf("interfaces", "duplicate interface name %q", iface.Name)
		}
		ifaceNames[iface.Name] = true
		if iface.Contract == "" {
			b.errf(fmt.Sprintf("interfaces[%q].contract", iface.Name), "required")
		}
	}
	scopeIDs := map[string]bool{}
	for i, sc := range s.Scopes {
		fp := fmt.Sprintf("scopes[%d]", i)
		if sc.ID == "" {
			b.errf(fp+".id", "required")
		} else {
			if scopeIDs[sc.ID] {
				b.errf(fp+".id", "duplicate scope id %q", sc.ID)
			}
			scopeIDs[sc.ID] = true
		}
		if sc.Title == "" {
			b.errf(fp+".title", "required")
		}
		if !scopeStatuses[sc.Status] {
			b.errf(fp+".status", "illegal status %q", sc.Status)
		}
		if sc.Hill == nil {
			b.errf(fp+".hill", "required")
		} else {
			h := *sc.Hill
			if h < 0.0 || h > 1.0 {
				b.errf(fp+".hill", "out of range: %v", h)
			}
			// hill: 1.0 iff status done
			if h == 1.0 && sc.Status != ScopeDone {
				b.errf(fp+".hill", "hill 1.0 requires status done")
			}
			if sc.Status == ScopeDone && h != 1.0 {
				b.errf(fp+".hill", "status done requires hill 1.0")
			}
		}
		for _, name := range sc.Interfaces {
			if !ifaceNames[name] {
				b.warnf(fp+".interfaces", "references unknown interface %q", name)
			}
		}
	}
	return &s, b.errs
}

func parseDecision(path string, fm []byte, body string) (*Decision, []DocError) {
	var d Decision
	b := unmarshal(path, fm, &d)
	if b.hasFatal() {
		return &d, b.errs
	}
	d.Path, d.Body = path, body

	req(b, "id", d.ID)
	if d.ID != "" && !ValidADRID(d.ID) {
		b.errf("id", "malformed adr id %q", d.ID)
	}
	req(b, "title", d.Title)
	req(b, "decided", d.Decided)
	if !decLayers[d.Layer] {
		b.errf("layer", "illegal layer %q", d.Layer)
	}
	if !decStatuses[d.Status] {
		b.errf("status", "illegal status %q", d.Status)
	}
	if !revs[d.Reversibility] {
		b.errf("reversibility", "illegal reversibility %q", d.Reversibility)
	}
	// supersession self-consistency (chain consistency is index-level)
	if d.SupersededBy != nil && *d.SupersededBy != "" {
		if d.Status != DecisionSuperseded {
			b.errf("superseded_by", "non-null superseded_by requires status superseded")
		}
		if !ValidADRID(*d.SupersededBy) {
			b.errf("superseded_by", "malformed adr id %q", *d.SupersededBy)
		}
	}
	if d.Status == DecisionSuperseded && (d.SupersededBy == nil || *d.SupersededBy == "") {
		b.errf("superseded_by", "status superseded requires a non-null superseded_by")
	}
	if d.Supersedes != nil && *d.Supersedes != "" && !ValidADRID(*d.Supersedes) {
		b.errf("supersedes", "malformed adr id %q", *d.Supersedes)
	}
	if d.Lineage != "" && !ValidLineageID(d.Lineage) {
		b.warnf("lineage", "malformed lineage id %q", d.Lineage)
	}
	return &d, b.errs
}

func parseCycle(path string, fm []byte, body string) (*Cycle, []DocError) {
	var c Cycle
	b := unmarshal(path, fm, &c)
	if b.hasFatal() {
		return &c, b.errs
	}
	c.Path, c.Body = path, body

	req(b, "id", c.ID)
	if c.ID != "" && !ValidCycleID(c.ID) {
		b.errf("id", "malformed cycle id %q", c.ID)
	}
	req(b, "starts", c.Starts)
	req(b, "ends", c.Ends)
	if !cycleStatuses[c.Status] {
		b.errf("status", "illegal status %q", c.Status)
	}
	if c.AppetiteWeeks <= 0 {
		b.errf("appetite_weeks", "must be positive")
	}
	if c.Capacity < 0 {
		b.errf("capacity", "must not be negative")
	}
	if len(c.Bets) > c.CapacityLimit() {
		b.errf("bets", "capacity breached: %d bets exceed capacity %d", len(c.Bets), c.CapacityLimit())
	}
	if ValidDate(c.Starts) && ValidDate(c.Ends) && c.Ends <= c.Starts {
		b.errf("ends", "must be after starts")
	}
	// a lineage must not be both bet and shelved in one cycle
	inBets := map[string]bool{}
	for i, bet := range c.Bets {
		if !ValidLineageID(bet.Lineage) {
			b.errf(fmt.Sprintf("bets[%d].lineage", i), "malformed lineage id %q", bet.Lineage)
		}
		inBets[bet.Lineage] = true
		if !appetites[bet.Appetite] {
			b.errf(fmt.Sprintf("bets[%d].appetite", i), "illegal appetite %q", bet.Appetite)
		}
	}
	for i, sh := range c.Shelved {
		if inBets[sh.Lineage] {
			b.errf(fmt.Sprintf("shelved[%d].lineage", i), "lineage %q is both bet and shelved", sh.Lineage)
		}
	}
	return &c, b.errs
}

// ParseManifest parses a pure-YAML manifest.yaml (no markdown body).
func ParseManifest(path string, content []byte) (*Manifest, []DocError) {
	var m Manifest
	b := &errbag{path: path}
	if err := yaml.Unmarshal(content, &m); err != nil {
		b.errf("", "invalid YAML: %s", err.Error())
		return &m, b.errs
	}
	m.Path = path
	if m.Version != 1 {
		b.errf("version", "unsupported schema version %d (want 1)", m.Version)
	}
	req(b, "project", m.Project)
	req(b, "project_id", m.ProjectID)
	if m.ProjectID != "" && !ValidProjectID(m.ProjectID) {
		b.errf("project_id", "malformed project id %q", m.ProjectID)
	}
	if !baselineModes[m.Baseline.Mode] {
		b.errf("baseline.mode", "illegal mode %q", m.Baseline.Mode)
	}
	if m.Baseline.Mode == ModeSpine && m.Baseline.Ref == "" {
		b.errf("baseline.ref", "required when mode is spine")
	}
	return &m, b.errs
}
