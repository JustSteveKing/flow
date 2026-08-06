package desktop

import (
	"net/http"
	"sort"

	"github.com/juststeveking/flow/internal/core"
)

func sortCycles(p *core.ProjectIndex) []*core.Cycle {
	out := make([]*core.Cycle, 0, len(p.Cycles))
	for _, c := range p.Cycles {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID }) // newest first
	return out
}

func sortSpecs(p *core.ProjectIndex) []*core.Spec {
	out := make([]*core.Spec, 0, len(p.Specs))
	for _, s := range p.Specs {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Lineage < out[j].Lineage })
	return out
}

func sortDecisions(p *core.ProjectIndex) []*core.Decision {
	out := make([]*core.Decision, 0, len(p.Decisions))
	for _, d := range p.Decisions {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortPitches(p *core.ProjectIndex) []*core.Pitch {
	out := make([]*core.Pitch, 0, len(p.Pitches))
	for _, pt := range p.Pitches {
		out = append(out, pt)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Lineage < out[j].Lineage })
	return out
}

// ProjectDetail is the full per-project read model: everything in one project's
// `.flow/` tree, shaped for the drill-down panel. Read-only lens over the index.
type ProjectDetail struct {
	ProjectID string        `json:"projectId"`
	Name      string        `json:"name"`
	Root      string        `json:"root"`
	Baseline  string        `json:"baseline"`
	Cycles    []CycleDTO    `json:"cycles"`
	Specs     []SpecDTO     `json:"specs"`
	Decisions []DecisionDTO `json:"decisions"`
	Pitches   []PitchDTO    `json:"pitches"`
	Errors    []string      `json:"errors"`
}

type CycleDTO struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	AppetiteWeeks int       `json:"appetiteWeeks"`
	Capacity      int       `json:"capacity"`
	Starts        string    `json:"starts"`
	Ends          string    `json:"ends"`
	Bets          []BetItem `json:"bets"`
	Shelved       []string  `json:"shelved"`
}

type BetItem struct {
	Lineage  string `json:"lineage"`
	Appetite string `json:"appetite"`
	Placed   string `json:"placed"`
}

type SpecDTO struct {
	Lineage          string         `json:"lineage"`
	Status           string         `json:"status"`
	Cycle            string         `json:"cycle"`
	NoGos            []string       `json:"noGos"`
	AssumesDecisions []string       `json:"assumesDecisions"`
	Interfaces       []InterfaceDTO `json:"interfaces"`
	Scopes           []ScopeDTO     `json:"scopes"`
}

type InterfaceDTO struct {
	Name     string `json:"name"`
	Contract string `json:"contract"`
}

type ScopeDTO struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Hill       float64  `json:"hill"`
	Interfaces []string `json:"interfaces"`
}

type DecisionDTO struct {
	ID            string `json:"id"`
	Layer         string `json:"layer"`
	Status        string `json:"status"`
	Title         string `json:"title"`
	Reversibility string `json:"reversibility"`
	Lineage       string `json:"lineage"`
	Supersedes    string `json:"supersedes"`
	SupersededBy  string `json:"supersededBy"`
	Decided       string `json:"decided"`
}

type PitchDTO struct {
	Lineage          string   `json:"lineage"`
	Status           string   `json:"status"`
	Title            string   `json:"title"`
	Appetite         string   `json:"appetite"`
	Problem          string   `json:"problem"`
	NoGos            []string `json:"noGos"`
	AssumesDecisions []string `json:"assumesDecisions"`
	RabbitHoles      []string `json:"rabbitHoles"`
}

// ProjectDetail builds the detail model for one project, or false if unknown.
func (w *Workspace) ProjectDetail(projectID string) (ProjectDetail, bool) {
	idx := w.snapshot()
	if idx == nil {
		return ProjectDetail{}, false
	}
	p, ok := idx.Projects[projectID]
	if !ok {
		return ProjectDetail{}, false
	}
	d := ProjectDetail{ProjectID: projectID, Name: projectID, Root: p.Root}
	if p.Manifest != nil {
		d.Name = p.Manifest.Project
		d.Baseline = string(p.Manifest.Baseline.Mode)
	}

	for _, c := range sortCycles(p) {
		cd := CycleDTO{
			ID: c.ID, Status: string(c.Status), AppetiteWeeks: c.AppetiteWeeks,
			Capacity: c.CapacityLimit(), Starts: c.Starts, Ends: c.Ends,
		}
		for _, b := range c.Bets {
			cd.Bets = append(cd.Bets, BetItem{Lineage: b.Lineage, Appetite: string(b.Appetite), Placed: b.Placed})
		}
		for _, s := range c.Shelved {
			cd.Shelved = append(cd.Shelved, s.Lineage)
		}
		d.Cycles = append(d.Cycles, cd)
	}

	for _, s := range sortSpecs(p) {
		sd := SpecDTO{
			Lineage: s.Lineage, Status: string(s.Status), Cycle: s.Cycle,
			NoGos: s.NoGos, AssumesDecisions: s.AssumesDecisions,
		}
		for _, i := range s.Interfaces {
			sd.Interfaces = append(sd.Interfaces, InterfaceDTO{Name: i.Name, Contract: i.Contract})
		}
		for _, sc := range s.Scopes {
			h := 0.0
			if sc.Hill != nil {
				h = *sc.Hill
			}
			sd.Scopes = append(sd.Scopes, ScopeDTO{
				ID: sc.ID, Title: sc.Title, Status: string(sc.Status), Hill: h, Interfaces: sc.Interfaces,
			})
		}
		d.Specs = append(d.Specs, sd)
	}

	for _, dec := range sortDecisions(p) {
		d.Decisions = append(d.Decisions, DecisionDTO{
			ID: dec.ID, Layer: string(dec.Layer), Status: string(dec.Status),
			Title: dec.Title, Reversibility: string(dec.Reversibility), Lineage: dec.Lineage,
			Supersedes: deref(dec.Supersedes), SupersededBy: deref(dec.SupersededBy), Decided: dec.Decided,
		})
	}

	for _, pt := range sortPitches(p) {
		d.Pitches = append(d.Pitches, PitchDTO{
			Lineage: pt.Lineage, Status: string(pt.Status), Title: pt.Title,
			Appetite: string(pt.Appetite), Problem: pt.Problem, NoGos: pt.NoGos,
			AssumesDecisions: pt.AssumesDecisions, RabbitHoles: pt.RabbitHoles,
		})
	}

	for _, e := range p.Errors {
		d.Errors = append(d.Errors, e.Error())
	}
	return d, true
}

func (w *Workspace) handleProject(rw http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	d, ok := w.ProjectDetail(id)
	if !ok {
		writeJSON(rw, http.StatusNotFound, map[string]string{"error": "unknown project " + id})
		return
	}
	writeJSON(rw, http.StatusOK, d)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
