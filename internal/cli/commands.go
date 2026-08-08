package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/juststeveking/flow/internal/core"
	"github.com/spf13/cobra"
)

// flow init
func cmdInit() *cobra.Command {
	var name, id string
	c := &cobra.Command{
		Use:   "init",
		Short: "Create a .flow/ tree and manifest in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if _, ok := core.FindRoot(cwd); ok {
				return fmt.Errorf("a flow project already exists at or above %s", cwd)
			}
			if id == "" {
				return fmt.Errorf("--id owner/name is required")
			}
			if !core.ValidProjectID(id) {
				return fmt.Errorf("malformed project id %q (want owner/name)", id)
			}
			if name == "" {
				name = filepath.Base(cwd)
			}
			for _, d := range []string{
				filepath.Join(cwd, ".flow", "pitches"),
				filepath.Join(cwd, ".flow", "specs"),
				filepath.Join(cwd, ".flow", "cycles"),
				filepath.Join(cwd, ".flow", "tasks"),
				filepath.Join(cwd, ".flow", "decisions", "baseline"),
				filepath.Join(cwd, ".flow", "decisions", "local"),
			} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					return err
				}
			}
			if err := scaffoldTaskAgents(cwd); err != nil {
				return err
			}
			if err := ensureGitattributes(cwd); err != nil {
				return err
			}
			m := &core.Manifest{
				Version: 1, Project: name, ProjectID: id,
				Baseline: core.Baseline{Mode: core.ModeVendored}, IDPrefix: "adr",
			}
			return write(cwd, m, "flow init "+id)
		},
	}
	c.Flags().StringVar(&name, "name", "", "human project name (defaults to directory name)")
	c.Flags().StringVar(&id, "id", "", "project id, owner/name (required)")
	return c
}

// flow shape <title>
func cmdShape() *cobra.Command {
	var appetite string
	c := &cobra.Command{
		Use:   "shape <title>",
		Short: "Create a pitch and open shaping",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := joinArgs(args)
			ap := core.Appetite(appetite)
			if ap != core.SmallBatch && ap != core.BigBatch {
				return fmt.Errorf("appetite must be small-batch or big-batch")
			}
			root, idx, err := loadProject()
			if err != nil {
				return err
			}
			ym, today := clock()
			lineage := core.MintLineage(ym, title, idx.Pitches)
			p := &core.Pitch{
				Lineage: lineage, Type: core.TypePitch, Status: core.PitchShaping,
				Title: title, Appetite: ap, Created: today, Updated: today,
				Body: "# " + title + "\n\nProblem, solution sketch, no-gos, rabbit holes.\n",
			}
			fmt.Println("lineage:", lineage)
			return write(root, p, "shape: "+lineage)
		},
	}
	c.Flags().StringVar(&appetite, "appetite", "big-batch", "small-batch or big-batch")
	return c
}

// flow table [--add <lineage>] [--shelve <lineage>]
func cmdTable() *cobra.Command {
	var add, shelve string
	c := &cobra.Command{
		Use:   "table",
		Short: "Show the betting table, or place/shelve a bet",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, idx, err := loadProject()
			if err != nil {
				return err
			}
			cycle, err := activeCycle(idx)
			if err != nil {
				return err
			}
			ym, today := clock()
			_ = ym

			switch {
			case add != "":
				p := idx.Pitches[add]
				if p == nil {
					return fmt.Errorf("no pitch %q", add)
				}
				// shaping -> shaped -> bet (place under the capacity budget)
				if err := core.TransitionPitch(p, core.PitchShaped); err != nil {
					return err
				}
				if err := core.TransitionPitch(p, core.PitchBet); err != nil {
					return err
				}
				if err := cycle.PlaceBet(add, p.Appetite, today); err != nil {
					return err
				}
				p.Updated = today
				return writeMany(root, "bet: "+add+" -> "+cycle.ID, cycle, p)
			case shelve != "":
				p := idx.Pitches[shelve]
				if p == nil {
					return fmt.Errorf("no pitch %q", shelve)
				}
				if err := core.TransitionPitch(p, core.PitchShelved); err != nil {
					return err
				}
				cycle.Shelve(shelve)
				p.Updated = today
				return writeMany(root, "shelve: "+shelve, cycle, p)
			default:
				printTable(idx, cycle)
				return nil
			}
		},
	}
	c.Flags().StringVar(&add, "add", "", "place a bet on this lineage (enforces capacity)")
	c.Flags().StringVar(&shelve, "shelve", "", "shelve this lineage")
	return c
}

// flow spec --from <lineage>
func cmdSpec() *cobra.Command {
	var from string
	c := &cobra.Command{
		Use:   "spec --from <lineage>",
		Short: "Scaffold a spec from a placed bet",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				return fmt.Errorf("--from <lineage> is required")
			}
			root, idx, err := loadProject()
			if err != nil {
				return err
			}
			p := idx.Pitches[from]
			if p == nil {
				return fmt.Errorf("no pitch %q", from)
			}
			if p.Status != core.PitchBet {
				return fmt.Errorf("pitch %q is %s, must be bet (run `flow table --add`)", from, p.Status)
			}
			if _, exists := idx.Specs[from]; exists {
				return fmt.Errorf("spec for %q already exists", from)
			}
			cycle, err := activeCycle(idx)
			if err != nil {
				return err
			}
			_, today := clock()
			s := &core.Spec{
				Lineage: from, Type: core.TypeSpec, Status: core.SpecSpecifying,
				Cycle: cycle.ID, NoGos: p.NoGos, AssumesDecisions: p.AssumesDecisions,
				Scopes:  []core.Scope{{ID: "scope-core", Title: p.Title, Status: core.ScopeUphill, Hill: f(0.0)}},
				Created: today, Updated: today,
				Body: "# Spec: " + p.Title + "\n\nCut scopes, lock interfaces and no-gos before building.\n",
			}
			return write(root, s, "spec: "+from)
		},
	}
	c.Flags().StringVar(&from, "from", "", "lineage of the bet to spec")
	return c
}

// flow decide <title> | --accept <id> | --supersede <old> --by <new>
func cmdDecide() *cobra.Command {
	var reversibility, layer, lineage, accept, supersede, by string
	c := &cobra.Command{
		Use:   "decide <title>",
		Short: "Raise or advance an ADR",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, idx, err := loadProject()
			if err != nil {
				return err
			}
			_, today := clock()

			switch {
			case accept != "":
				d := idx.Decisions[accept]
				if d == nil {
					return fmt.Errorf("no decision %q", accept)
				}
				if err := core.TransitionDecision(d, core.DecisionAccepted); err != nil {
					return err
				}
				return write(root, d, "accept: "+accept)
			case supersede != "":
				if by == "" {
					return fmt.Errorf("--supersede needs --by <new-id>")
				}
				older, newer := idx.Decisions[supersede], idx.Decisions[by]
				if older == nil {
					return fmt.Errorf("no decision %q", supersede)
				}
				if newer == nil {
					return fmt.Errorf("no decision %q", by)
				}
				if err := core.Supersede(older, newer); err != nil {
					return err
				}
				return writeMany(root, "supersede: "+supersede+" by "+by, older, newer)
			default:
				if len(args) == 0 {
					return fmt.Errorf("a title, --accept, or --supersede is required")
				}
				title := joinArgs(args)
				rev := core.Reversibility(reversibility)
				if rev != core.OneWay && rev != core.Reversible {
					return fmt.Errorf("--reversibility must be one-way or reversible")
				}
				id := core.NextADRID(idx.Manifest.IDPrefix, idx.Decisions, title)
				d := &core.Decision{
					ID: id, Type: core.TypeDecision, Layer: core.DecisionLayer(layer),
					Status: core.DecisionProposed, Title: title, Lineage: lineage,
					Reversibility: rev, Decided: today,
					Body: "## Context\n\n## Decision\n\n## Consequences\n",
				}
				fmt.Println("id:", id)
				return write(root, d, "decide: "+id)
			}
		},
	}
	c.Flags().StringVar(&reversibility, "reversibility", "reversible", "one-way or reversible")
	c.Flags().StringVar(&layer, "layer", "local", "local or baseline")
	c.Flags().StringVar(&lineage, "lineage", "", "the work chain this decision serves")
	c.Flags().StringVar(&accept, "accept", "", "accept a proposed decision by id")
	c.Flags().StringVar(&supersede, "supersede", "", "supersede an older decision by id")
	c.Flags().StringVar(&by, "by", "", "the superseding decision id")
	return c
}

// flow build --start <lineage> | --hill <scope>=<value> --spec <lineage>
func cmdBuild() *cobra.Command {
	var start, spec, hill string
	c := &cobra.Command{
		Use:   "build --start <lineage>",
		Short: "Lock a spec into building, or update a scope's hill",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, idx, err := loadProject()
			if err != nil {
				return err
			}
			_, today := clock()

			if start != "" {
				s := idx.Specs[start]
				if s == nil {
					return fmt.Errorf("no spec %q", start)
				}
				if err := core.TransitionSpec(s, core.SpecBuilding); err != nil {
					return err
				}
				s.Updated = today
				return write(root, s, "build start: "+start)
			}
			if hill != "" {
				if spec == "" {
					return fmt.Errorf("--hill needs --spec <lineage>")
				}
				s := idx.Specs[spec]
				if s == nil {
					return fmt.Errorf("no spec %q", spec)
				}
				scopeID, val, err := parseHill(hill)
				if err != nil {
					return err
				}
				found := false
				for i := range s.Scopes {
					if s.Scopes[i].ID == scopeID {
						s.Scopes[i].Hill = f(val)
						s.Scopes[i].Status = hillStatus(val)
						found = true
					}
				}
				if !found {
					return fmt.Errorf("no scope %q in spec %q", scopeID, spec)
				}
				s.Updated = today
				return write(root, s, fmt.Sprintf("hill: %s %s=%.2f", spec, scopeID, val))
			}
			return fmt.Errorf("--start or --hill is required")
		},
	}
	c.Flags().StringVar(&start, "start", "", "lock interfaces/no-gos and begin building this spec")
	c.Flags().StringVar(&spec, "spec", "", "spec lineage for --hill")
	c.Flags().StringVar(&hill, "hill", "", "update a scope hill, form scope-id=0.5")
	return c
}

// flow archive --cooldown <cycle> | --close <cycle>
func cmdArchive() *cobra.Command {
	var cooldown, closeCycle string
	c := &cobra.Command{
		Use:   "archive",
		Short: "Enter cooldown or close a cycle (freeze specs, finalise ADRs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, idx, err := loadProject()
			if err != nil {
				return err
			}
			_, today := clock()

			if cooldown != "" {
				c := idx.Cycles[cooldown]
				if c == nil {
					return fmt.Errorf("no cycle %q", cooldown)
				}
				if err := core.TransitionCycle(c, core.CycleCooldown); err != nil {
					return err
				}
				return write(root, c, "cooldown: "+cooldown)
			}
			if closeCycle != "" {
				c := idx.Cycles[closeCycle]
				if c == nil {
					return fmt.Errorf("no cycle %q", closeCycle)
				}
				if err := core.TransitionCycle(c, core.CycleClosed); err != nil {
					return err
				}
				docs := []any{c}
				// freeze specs in this cycle
				for _, s := range idx.Specs {
					if s.Cycle == closeCycle && s.Status == core.SpecBuilding {
						if err := core.TransitionSpec(s, core.SpecFrozen); err != nil {
							return err
						}
						s.Updated = today
						docs = append(docs, s)
					}
				}
				// finalise proposed ADRs emitted by this cycle's lineages
				lineages := map[string]bool{}
				for _, s := range idx.Specs {
					if s.Cycle == closeCycle {
						lineages[s.Lineage] = true
					}
				}
				for _, d := range idx.Decisions {
					if d.Status == core.DecisionProposed && lineages[d.Lineage] {
						if err := core.TransitionDecision(d, core.DecisionAccepted); err != nil {
							return err
						}
						docs = append(docs, d)
					}
				}
				return writeMany(root, "close: "+closeCycle, docs...)
			}
			return fmt.Errorf("--cooldown or --close is required")
		},
	}
	c.Flags().StringVar(&cooldown, "cooldown", "", "put this cycle into cooldown")
	c.Flags().StringVar(&closeCycle, "close", "", "close this cycle and archive its work")
	return c
}

// flow status
func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show a read-only summary of the project",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, idx, err := loadProject()
			if err != nil {
				return err
			}
			printStatus(idx)
			return nil
		},
	}
}
