package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/juststeveking/flow/internal/core"
)

func joinArgs(args []string) string { return strings.Join(args, " ") }

func f(v float64) *float64 { return &v }

// parseHill parses a "scope-id=0.5" argument.
func parseHill(s string) (scope string, val float64, err error) {
	i := strings.LastIndex(s, "=")
	if i < 0 {
		return "", 0, fmt.Errorf("--hill form is scope-id=0.5")
	}
	scope = s[:i]
	val, err = strconv.ParseFloat(s[i+1:], 64)
	if err != nil {
		return "", 0, fmt.Errorf("bad hill value: %v", err)
	}
	if val < 0 || val > 1 {
		return "", 0, fmt.Errorf("hill must be in [0.0, 1.0]")
	}
	return scope, val, nil
}

// hillStatus maps a hill position to a scope status.
func hillStatus(v float64) core.ScopeStatus {
	switch {
	case v >= 1.0:
		return core.ScopeDone
	case v >= 0.5:
		return core.ScopeDownhill
	default:
		return core.ScopeUphill
	}
}

func printTable(idx *core.ProjectIndex, c *core.Cycle) {
	fmt.Printf("cycle %s (%s)  capacity %d/%d  ends %s\n", c.ID, c.Status, len(c.Bets), c.CapacityLimit(), c.Ends)
	fmt.Println("bets:")
	if len(c.Bets) == 0 {
		fmt.Println("  (none)")
	}
	for _, b := range c.Bets {
		fmt.Printf("  * %-28s %-11s %s\n", b.Lineage, b.Appetite, titleFor(idx, b.Lineage))
	}
	if len(c.Shelved) > 0 {
		fmt.Println("shelved:")
		for _, s := range c.Shelved {
			fmt.Printf("  - %-28s %s\n", s.Lineage, titleFor(idx, s.Lineage))
		}
	}
}

func printStatus(idx *core.ProjectIndex) {
	fmt.Printf("project %s (%s)\n", idx.Manifest.Project, idx.Manifest.ProjectID)

	cycles := sortedCycles(idx)
	for _, c := range cycles {
		fmt.Printf("\ncycle %s [%s]  bets %d/%d  %s..%s\n", c.ID, c.Status, len(c.Bets), c.CapacityLimit(), c.Starts, c.Ends)
	}

	if len(idx.Specs) > 0 {
		fmt.Println("\nspecs:")
		for _, s := range sortedSpecs(idx) {
			fmt.Printf("  %s [%s] cycle %s\n", s.Lineage, s.Status, s.Cycle)
			for _, sc := range s.Scopes {
				h := 0.0
				if sc.Hill != nil {
					h = *sc.Hill
				}
				fmt.Printf("    %-16s %-9s hill %.2f\n", sc.ID, sc.Status, h)
			}
		}
	}

	heads := idx.ADRHeads()
	if len(heads) > 0 {
		fmt.Println("\ndecisions (heads):")
		for _, d := range heads {
			fmt.Printf("  %-30s [%s] %s\n", d.ID, d.Status, d.Title)
		}
	}

	var fatal, warn int
	for _, e := range idx.Errors {
		if e.Severity == core.SevError {
			fatal++
		} else {
			warn++
		}
	}
	if fatal+warn > 0 {
		fmt.Printf("\nindex: %d errors, %d warnings\n", fatal, warn)
		for _, e := range idx.Errors {
			fmt.Println("  ", e.Error())
		}
	}
}

func titleFor(idx *core.ProjectIndex, lineage string) string {
	if p, ok := idx.Pitches[lineage]; ok {
		return p.Title
	}
	return ""
}

func sortedCycles(idx *core.ProjectIndex) []*core.Cycle {
	out := make([]*core.Cycle, 0, len(idx.Cycles))
	for _, c := range idx.Cycles {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedSpecs(idx *core.ProjectIndex) []*core.Spec {
	out := make([]*core.Spec, 0, len(idx.Specs))
	for _, s := range idx.Specs {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Lineage < out[j].Lineage })
	return out
}
