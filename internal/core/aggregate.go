package core

import "sort"

// Index aggregates many ProjectIndexes on one plane, keyed by project id.
// Reads span all roots; identity is project-qualified so two projects cannot
// collide. It is a rebuildable projection, never authoritative.
type Index struct {
	Projects map[string]*ProjectIndex // by project id
}

// NewIndex builds an aggregated index from a set of already-loaded projects.
// Projects with a missing or malformed project id are skipped and reported.
func NewIndex(projects ...*ProjectIndex) (*Index, []DocError) {
	idx := &Index{Projects: map[string]*ProjectIndex{}}
	var errs []DocError
	for _, p := range projects {
		pid := p.ProjectID()
		if pid == "" {
			errs = append(errs, DocError{Path: p.Root, Severity: SevError, Message: "project has no project_id; skipped from aggregate"})
			continue
		}
		if existing, clash := idx.Projects[pid]; clash {
			errs = append(errs, DocError{Path: p.Root, Severity: SevError, Message: "project_id " + pid + " collides with " + existing.Root})
			continue
		}
		idx.Projects[pid] = p
	}
	return idx, errs
}

// Resolve maps a qualified id (project_id:local_id) back to its owning project.
// Every write routes through this: a write always resolves to exactly one project.
func (i *Index) Resolve(qualified string) (*ProjectIndex, string, bool) {
	pid, local, ok := splitQualified(qualified)
	if !ok {
		return nil, "", false
	}
	p, ok := i.Projects[pid]
	if !ok {
		return nil, "", false
	}
	return p, local, true
}

func splitQualified(q string) (projectID, localID string, ok bool) {
	// project id is owner/name, local id follows the last colon.
	for idx := len(q) - 1; idx >= 0; idx-- {
		if q[idx] == ':' {
			return q[:idx], q[idx+1:], idx > 0 && idx < len(q)-1
		}
	}
	return "", "", false
}

// BettingTableRow is one bet or shelved pitch on the unified betting table.
type BettingTableRow struct {
	ProjectID string
	Cycle     string
	Lineage   string
	Title     string
	Appetite  Appetite
	Shelved   bool
}

// BettingTable aggregates every active cycle's bets and shelved pitches across
// all projects, for the desktop's unified betting table view.
func (i *Index) BettingTable() []BettingTableRow {
	var rows []BettingTableRow
	for pid, p := range i.Projects {
		for _, c := range p.Cycles {
			if c.Status != CycleActive {
				continue
			}
			for _, bet := range c.Bets {
				rows = append(rows, BettingTableRow{pid, c.ID, bet.Lineage, titleOf(p, bet.Lineage), bet.Appetite, false})
			}
			for _, sh := range c.Shelved {
				rows = append(rows, BettingTableRow{pid, c.ID, sh.Lineage, titleOf(p, sh.Lineage), "", true})
			}
		}
	}
	sort.Slice(rows, func(a, b int) bool {
		if rows[a].ProjectID != rows[b].ProjectID {
			return rows[a].ProjectID < rows[b].ProjectID
		}
		return rows[a].Lineage < rows[b].Lineage
	})
	return rows
}

// HillPoint is one building scope on the portfolio hill chart.
type HillPoint struct {
	ProjectID string
	Lineage   string
	ScopeID   string
	Title     string
	Status    ScopeStatus
	Hill      float64
}

// PortfolioHill aggregates every building scope across all projects onto one hill.
func (i *Index) PortfolioHill() []HillPoint {
	var pts []HillPoint
	for pid, p := range i.Projects {
		for _, s := range p.Specs {
			if s.Status != SpecBuilding {
				continue
			}
			for _, sc := range s.Scopes {
				h := 0.0
				if sc.Hill != nil {
					h = *sc.Hill
				}
				pts = append(pts, HillPoint{pid, s.Lineage, sc.ID, sc.Title, sc.Status, h})
			}
		}
	}
	sort.Slice(pts, func(a, b int) bool {
		if pts[a].ProjectID != pts[b].ProjectID {
			return pts[a].ProjectID < pts[b].ProjectID
		}
		if pts[a].Lineage != pts[b].Lineage {
			return pts[a].Lineage < pts[b].Lineage
		}
		return pts[a].ScopeID < pts[b].ScopeID
	})
	return pts
}

// SupersessionEdge is one directed edge (older superseded by newer) qualified by
// project, for the ADR supersession graph view.
type SupersessionEdge struct {
	ProjectID string
	From      string // older adr id
	To        string // newer adr id
	FromLayer DecisionLayer
	ToLayer   DecisionLayer
}

// SupersessionGraph aggregates every supersession edge across all projects.
func (i *Index) SupersessionGraph() []SupersessionEdge {
	var edges []SupersessionEdge
	for pid, p := range i.Projects {
		byID := map[string]*Decision{}
		for _, d := range p.AllADRs {
			byID[d.ID] = d
		}
		for _, d := range p.AllADRs {
			if d.SupersededBy != nil && *d.SupersededBy != "" {
				to := *d.SupersededBy
				var toLayer DecisionLayer
				if t, ok := byID[to]; ok {
					toLayer = t.Layer
				}
				edges = append(edges, SupersessionEdge{pid, d.ID, to, d.Layer, toLayer})
			}
		}
	}
	sort.Slice(edges, func(a, b int) bool {
		if edges[a].ProjectID != edges[b].ProjectID {
			return edges[a].ProjectID < edges[b].ProjectID
		}
		return edges[a].From < edges[b].From
	})
	return edges
}

func titleOf(p *ProjectIndex, lineage string) string {
	if pt, ok := p.Pitches[lineage]; ok {
		return pt.Title
	}
	return ""
}
