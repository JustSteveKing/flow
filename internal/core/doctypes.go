package core

// DocType discriminates the five flow document types.
type DocType string

const (
	TypePitch    DocType = "pitch"
	TypeSpec     DocType = "spec"
	TypeDecision DocType = "decision"
	TypeCycle    DocType = "cycle"
)

// Enumerations, frozen by SCHEMA.md schema version 1.
type (
	PitchStatus    string
	SpecStatus     string
	ScopeStatus    string
	DecisionStatus string
	DecisionLayer  string
	Reversibility  string
	CycleStatus    string
	Appetite       string
	BaselineMode   string
)

const (
	PitchShaping PitchStatus = "shaping"
	PitchBetting PitchStatus = "betting"
	PitchShaped  PitchStatus = "shaped"
	PitchBet     PitchStatus = "bet"
	PitchShelved PitchStatus = "shelved"

	SpecSpecifying SpecStatus = "specifying"
	SpecBuilding   SpecStatus = "building"
	SpecFrozen     SpecStatus = "frozen"

	ScopeUphill   ScopeStatus = "uphill"
	ScopeDownhill ScopeStatus = "downhill"
	ScopeDone     ScopeStatus = "done"

	DecisionProposed   DecisionStatus = "proposed"
	DecisionAccepted   DecisionStatus = "accepted"
	DecisionSuperseded DecisionStatus = "superseded"

	LayerBaseline DecisionLayer = "baseline"
	LayerLocal    DecisionLayer = "local"

	OneWay     Reversibility = "one-way"
	Reversible Reversibility = "reversible"

	CycleActive   CycleStatus = "active"
	CycleCooldown CycleStatus = "cooldown"
	CycleClosed   CycleStatus = "closed"

	SmallBatch Appetite = "small-batch"
	BigBatch   Appetite = "big-batch"

	ModeSpine    BaselineMode = "spine"
	ModeVendored BaselineMode = "vendored"
)

// membership sets for enum validation
var (
	pitchStatuses = set(PitchShaping, PitchBetting, PitchShaped, PitchBet, PitchShelved)
	specStatuses  = set(SpecSpecifying, SpecBuilding, SpecFrozen)
	scopeStatuses = set(ScopeUphill, ScopeDownhill, ScopeDone)
	decStatuses   = set(DecisionProposed, DecisionAccepted, DecisionSuperseded)
	decLayers     = set(LayerBaseline, LayerLocal)
	revs          = set(OneWay, Reversible)
	cycleStatuses = set(CycleActive, CycleCooldown, CycleClosed)
	appetites     = set(SmallBatch, BigBatch)
	baselineModes = set(ModeSpine, ModeVendored)
)

func set[T comparable](vals ...T) map[T]bool {
	m := make(map[T]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// Pitch is a Shape Up pitch: the container-level unit of work.
type Pitch struct {
	Lineage          string      `yaml:"lineage"`
	Type             DocType     `yaml:"type"`
	Status           PitchStatus `yaml:"status"`
	Title            string      `yaml:"title"`
	Appetite         Appetite    `yaml:"appetite"`
	Problem          string      `yaml:"problem,omitempty"`
	NoGos            []string    `yaml:"no_gos,omitempty"`
	AssumesDecisions []string    `yaml:"assumes_decisions,omitempty"`
	RabbitHoles      []string    `yaml:"rabbit_holes,omitempty"`
	Created          string      `yaml:"created"`
	Updated          string      `yaml:"updated"`

	Path string `yaml:"-"` // absolute source path, set by the loader
	Body string `yaml:"-"` // markdown after the frontmatter
}

// Interface is a fixed contract a scope must honour (locked at build start).
type Interface struct {
	Name     string `yaml:"name"`
	Contract string `yaml:"contract"`
}

// Scope is a slice of buildable work with a hill position.
type Scope struct {
	ID         string      `yaml:"id"`
	Title      string      `yaml:"title"`
	Status     ScopeStatus `yaml:"status"`
	Hill       *float64    `yaml:"hill"` // pointer: distinguishes missing from 0.0
	Interfaces []string    `yaml:"interfaces,omitempty"`
}

// Spec is the buildable surface inside a cycle.
type Spec struct {
	Lineage          string      `yaml:"lineage"`
	Type             DocType     `yaml:"type"`
	Status           SpecStatus  `yaml:"status"`
	Cycle            string      `yaml:"cycle"`
	AssumesDecisions []string    `yaml:"assumes_decisions,omitempty"`
	NoGos            []string    `yaml:"no_gos,omitempty"`
	Interfaces       []Interface `yaml:"interfaces,omitempty"`
	Scopes           []Scope     `yaml:"scopes"`
	Created          string      `yaml:"created"`
	Updated          string      `yaml:"updated"`

	Path string `yaml:"-"`
	Body string `yaml:"-"`
}

// Decision is an Architecture Decision Record.
type Decision struct {
	ID            string         `yaml:"id"`
	Type          DocType        `yaml:"type"`
	Layer         DecisionLayer  `yaml:"layer"`
	Status        DecisionStatus `yaml:"status"`
	Title         string         `yaml:"title"`
	Lineage       string         `yaml:"lineage,omitempty"`
	Reversibility Reversibility  `yaml:"reversibility"`
	Supersedes    *string        `yaml:"supersedes"`
	SupersededBy  *string        `yaml:"superseded_by"`
	Decided       string         `yaml:"decided"`

	Path string `yaml:"-"`
	Body string `yaml:"-"`
}

// Bet binds a pitch to a cycle.
type Bet struct {
	Lineage  string   `yaml:"lineage"`
	Appetite Appetite `yaml:"appetite"`
	Placed   string   `yaml:"placed"`
}

// Shelved records a pitch passed over in a cycle.
type Shelved struct {
	Lineage string `yaml:"lineage"`
}

// Cycle is a fixed-appetite Shape Up cycle holding the betting table.
type Cycle struct {
	ID            string      `yaml:"id"`
	Type          DocType     `yaml:"type"`
	Status        CycleStatus `yaml:"status"`
	AppetiteWeeks int         `yaml:"appetite_weeks"`
	Capacity      int         `yaml:"capacity,omitempty"` // max concurrent bets; 0 means default (1)
	Starts        string      `yaml:"starts"`
	Ends          string      `yaml:"ends"`
	Bets          []Bet       `yaml:"bets,omitempty"`
	Shelved       []Shelved   `yaml:"shelved,omitempty"`

	Path string `yaml:"-"`
	Body string `yaml:"-"`
}

// Baseline configures where durable cross-project ADRs live.
type Baseline struct {
	Mode BaselineMode `yaml:"mode"`
	Ref  string       `yaml:"ref,omitempty"`
}

// Manifest is the per-project configuration, read first by the core.
type Manifest struct {
	Version   int      `yaml:"version"`
	Project   string   `yaml:"project"`
	ProjectID string   `yaml:"project_id"`
	Baseline  Baseline `yaml:"baseline"`
	IDPrefix  string   `yaml:"id_prefix,omitempty"`

	Path string `yaml:"-"`
}
