# flow sub-agents

Each agent proposes and stops at a gate. None walks through a gate; a person
commits the state transition (PLAN.md section 4).

| Agent | Trigger (state) | Produces | Stops before |
|-------|-----------------|----------|--------------|
| `flow-shaping-critic` | pitch `shaping` | `reviews/<lineage>-shaping-critic.md` | betting (`flow table --add`) |
| `flow-spec-proposer` | pitch `bet`, no spec | draft `specs/<lineage>.md` (specifying) | building (`flow build --start`) |
| `flow-implementer` | spec `building`, one scope | code + that scope's hill update | freeze / other scopes / cycle moves |
| `flow-spec-reviewer` | scope `done` | `reviews/<lineage>-spec-reviewer.md` | merge and freeze |
| `flow-archiver` | cycle `cooldown` | `reviews/<cycle-id>-archiver.md` freeze proposal | close (`flow archive --close`) |

All drive the `flow` CLI. The gate verbs themselves stay in human hands.
