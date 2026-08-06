package core

import "fmt"

// Transitions encode the state machine from PLAN.md section 2. Each map lists the
// statuses reachable from a given status. Guards beyond mere reachability (for
// example locking interfaces on the build transition) are enforced separately.

var pitchTransitions = map[PitchStatus]map[PitchStatus]bool{
	PitchShaping: {PitchShaped: true},                      // flow table --add
	PitchShaped:  {PitchBet: true, PitchShelved: true},     // bet, or shelve
	PitchBet:     {PitchShaping: true, PitchShelved: true}, // circuit breaker: re-shape
	PitchShelved: {PitchShaping: true},                     // revive to re-shape
}

var specTransitions = map[SpecStatus]map[SpecStatus]bool{
	SpecSpecifying: {SpecBuilding: true},
	SpecBuilding:   {SpecFrozen: true},
	SpecFrozen:     {}, // terminal
}

var decisionTransitions = map[DecisionStatus]map[DecisionStatus]bool{
	DecisionProposed:   {DecisionAccepted: true},
	DecisionAccepted:   {DecisionSuperseded: true},
	DecisionSuperseded: {}, // terminal
}

var cycleTransitions = map[CycleStatus]map[CycleStatus]bool{
	CycleActive:   {CycleCooldown: true},
	CycleCooldown: {CycleClosed: true},
	CycleClosed:   {}, // terminal
}

// IllegalTransitionError reports a rejected state change.
type IllegalTransitionError struct {
	Type DocType
	From string
	To   string
}

func (e IllegalTransitionError) Error() string {
	return fmt.Sprintf("illegal %s transition %s -> %s", e.Type, e.From, e.To)
}

// TransitionPitch moves a pitch to a new status if the edge is legal.
func TransitionPitch(p *Pitch, to PitchStatus) error {
	if p.Status == to {
		return nil
	}
	if !pitchTransitions[p.Status][to] {
		return IllegalTransitionError{TypePitch, string(p.Status), string(to)}
	}
	p.Status = to
	return nil
}

// TransitionSpec moves a spec to a new status. The specifying -> building edge is
// guarded: interfaces MUST be locked (non-empty) before building starts, per
// Decision 1 (interfaces and no-gos fixed, scope bodies just-in-time).
func TransitionSpec(s *Spec, to SpecStatus) error {
	if s.Status == to {
		return nil
	}
	if !specTransitions[s.Status][to] {
		return IllegalTransitionError{TypeSpec, string(s.Status), string(to)}
	}
	if to == SpecBuilding && len(s.Interfaces) == 0 {
		return fmt.Errorf("cannot start building: interfaces must be locked first (Decision 1)")
	}
	s.Status = to
	return nil
}

// TransitionDecision walks the ADR lifecycle proposed -> accepted -> superseded.
// Superseding requires the id of the superseding record and maintains both ends
// of the chain on the two documents.
func TransitionDecision(d *Decision, to DecisionStatus) error {
	if d.Status == to {
		return nil
	}
	if !decisionTransitions[d.Status][to] {
		return IllegalTransitionError{TypeDecision, string(d.Status), string(to)}
	}
	d.Status = to
	return nil
}

// Supersede marks older superseded by newer and links both ends of the chain.
// Both records must then be written. newer must already exist as an accepted ADR.
func Supersede(older, newer *Decision) error {
	if older.ID == newer.ID {
		return fmt.Errorf("a decision cannot supersede itself")
	}
	if older.Layer == LayerBaseline && newer.Layer == LayerLocal {
		// permitted: the local supersession escape hatch (Decision 4)
	}
	if newer.Layer == LayerBaseline && older.Layer == LayerLocal {
		return fmt.Errorf("a baseline record must not supersede a local record (Decision 4)")
	}
	older.Status = DecisionSuperseded
	older.SupersededBy = &newer.ID
	newer.Supersedes = &older.ID
	return nil
}

// TransitionCycle walks active -> cooldown -> closed (the circuit breaker path).
func TransitionCycle(c *Cycle, to CycleStatus) error {
	if c.Status == to {
		return nil
	}
	if !cycleTransitions[c.Status][to] {
		return IllegalTransitionError{TypeCycle, string(c.Status), string(to)}
	}
	c.Status = to
	return nil
}
