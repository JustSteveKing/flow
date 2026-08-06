package desktop

import (
	"encoding/json"
	"io/fs"
	"net/http"
)

// Handler builds the in-process HTTP surface the desktop shell loads: JSON under
// /api, static assets otherwise. Keeping the shell to a window over this handler
// means the app is a thin lens (adr-0001); the same handler is unit-testable
// without any GUI.
func (w *Workspace) Handler(static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/snapshot", w.handleSnapshot)
	mux.HandleFunc("/api/project", w.handleProject)
	mux.HandleFunc("/api/doc", w.handleDoc)
	mux.HandleFunc("/api/bet", w.handleBet)
	mux.HandleFunc("/api/shelve", w.handleShelve)
	mux.HandleFunc("/api/hill", w.handleHill)
	if static != nil {
		mux.Handle("/", http.FileServer(http.FS(static)))
	}
	return mux
}

// Snapshot is the whole read model the frontend renders: the three cross-project
// views plus the project registry panel.
type Snapshot struct {
	Projects      []ProjectSummary `json:"projects"`
	BettingTable  []BetRow         `json:"bettingTable"`
	PortfolioHill []HillDTO        `json:"portfolioHill"`
	Supersession  []EdgeDTO        `json:"supersession"`
}

type BetRow struct {
	ProjectID string `json:"projectId"`
	Cycle     string `json:"cycle"`
	Lineage   string `json:"lineage"`
	Title     string `json:"title"`
	Appetite  string `json:"appetite"`
	Shelved   bool   `json:"shelved"`
}

type HillDTO struct {
	ProjectID string  `json:"projectId"`
	Lineage   string  `json:"lineage"`
	ScopeID   string  `json:"scopeId"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	Hill      float64 `json:"hill"`
}

type EdgeDTO struct {
	ProjectID string `json:"projectId"`
	From      string `json:"from"`
	To        string `json:"to"`
	FromLayer string `json:"fromLayer"`
	ToLayer   string `json:"toLayer"`
}

func (w *Workspace) snapshotDTO() Snapshot {
	s := Snapshot{Projects: w.Projects()}
	for _, r := range w.BettingTable() {
		s.BettingTable = append(s.BettingTable, BetRow{
			ProjectID: r.ProjectID, Cycle: r.Cycle, Lineage: r.Lineage,
			Title: r.Title, Appetite: string(r.Appetite), Shelved: r.Shelved,
		})
	}
	for _, h := range w.PortfolioHill() {
		s.PortfolioHill = append(s.PortfolioHill, HillDTO{
			ProjectID: h.ProjectID, Lineage: h.Lineage, ScopeID: h.ScopeID,
			Title: h.Title, Status: string(h.Status), Hill: h.Hill,
		})
	}
	for _, e := range w.SupersessionGraph() {
		s.Supersession = append(s.Supersession, EdgeDTO{
			ProjectID: e.ProjectID, From: e.From, To: e.To,
			FromLayer: string(e.FromLayer), ToLayer: string(e.ToLayer),
		})
	}
	return s
}

func (w *Workspace) handleSnapshot(rw http.ResponseWriter, r *http.Request) {
	writeJSON(rw, http.StatusOK, w.snapshotDTO())
}

type routeReq struct {
	ProjectID string  `json:"projectId"`
	Lineage   string  `json:"lineage"`
	ScopeID   string  `json:"scopeId"`
	Hill      float64 `json:"hill"`
}

func (w *Workspace) handleBet(rw http.ResponseWriter, r *http.Request) {
	req, ok := decode(rw, r)
	if !ok {
		return
	}
	reply(rw, w.PlaceBet(req.ProjectID, req.Lineage))
}

func (w *Workspace) handleShelve(rw http.ResponseWriter, r *http.Request) {
	req, ok := decode(rw, r)
	if !ok {
		return
	}
	reply(rw, w.ShelveBet(req.ProjectID, req.Lineage))
}

func (w *Workspace) handleHill(rw http.ResponseWriter, r *http.Request) {
	req, ok := decode(rw, r)
	if !ok {
		return
	}
	reply(rw, w.SetHill(req.ProjectID, req.Lineage, req.ScopeID, req.Hill))
}

func decode(rw http.ResponseWriter, r *http.Request) (routeReq, bool) {
	if r.Method != http.MethodPost {
		writeJSON(rw, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return routeReq{}, false
	}
	var req routeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(rw, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return routeReq{}, false
	}
	return req, true
}

func reply(rw http.ResponseWriter, err error) {
	if err != nil {
		writeJSON(rw, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(rw, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(rw http.ResponseWriter, code int, v any) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	_ = json.NewEncoder(rw).Encode(v)
}
