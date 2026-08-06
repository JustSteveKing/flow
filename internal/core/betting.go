package core

import "fmt"

// CapacityLimit is the effective count-based appetite budget for the cycle.
// An unset capacity means the solo default of one concurrent bet (Decision 3).
func (c *Cycle) CapacityLimit() int {
	if c.Capacity <= 0 {
		return 1
	}
	return c.Capacity
}

// PlaceBet adds a bet to the cycle, enforcing the capacity budget. It refuses a
// bet that would breach capacity; the caller must shelve a pitch to make room.
// This is the solo forcing function (Decision 3): a hard, enforced cap.
func (c *Cycle) PlaceBet(lineage string, appetite Appetite, placed string) error {
	for _, b := range c.Bets {
		if b.Lineage == lineage {
			return fmt.Errorf("%s is already bet in %s", lineage, c.ID)
		}
	}
	if len(c.Bets) >= c.CapacityLimit() {
		return fmt.Errorf("cycle %s is at capacity (%d); shelve a bet to make room", c.ID, c.CapacityLimit())
	}
	// remove from shelved if it was there
	c.Unshelve(lineage)
	c.Bets = append(c.Bets, Bet{Lineage: lineage, Appetite: appetite, Placed: placed})
	return nil
}

// Shelve records a pitch as passed over in this cycle and removes any bet on it.
func (c *Cycle) Shelve(lineage string) {
	kept := c.Bets[:0]
	for _, b := range c.Bets {
		if b.Lineage != lineage {
			kept = append(kept, b)
		}
	}
	c.Bets = kept
	for _, s := range c.Shelved {
		if s.Lineage == lineage {
			return
		}
	}
	c.Shelved = append(c.Shelved, Shelved{Lineage: lineage})
}

// Unshelve removes a pitch from the shelved list, if present.
func (c *Cycle) Unshelve(lineage string) {
	kept := c.Shelved[:0]
	for _, s := range c.Shelved {
		if s.Lineage != lineage {
			kept = append(kept, s)
		}
	}
	c.Shelved = kept
}
