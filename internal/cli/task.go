package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/juststeveking/flow/internal/core"
	"github.com/juststeveking/flow/internal/task"
	"github.com/spf13/cobra"
)

// flagTaskJSON is the task tree's --json switch: structured output on stdout with
// human text suppressed, for agent consumption.
var flagTaskJSON bool

// cmdTask is the `flow task` parent. All logic lives in internal/task; these are
// thin callers (adr-0002).
func cmdTask() *cobra.Command {
	c := &cobra.Command{
		Use:   "task",
		Short: "Create, claim, and advance tasks",
	}
	c.PersistentFlags().BoolVar(&flagTaskJSON, "json", false, "emit structured JSON on stdout, suppress human text")
	c.AddCommand(
		cmdTaskNew(), cmdTaskList(), cmdTaskShow(), cmdTaskNext(),
		cmdTaskClaim(), cmdTaskRelease(), cmdTaskStatus(),
		cmdTaskCheck(), cmdTaskUncheck(), cmdTaskLog(), cmdTaskSync(),
	)
	return c
}

func loadTasks() (string, *task.TaskSet, error) {
	root, err := mustRoot()
	if err != nil {
		return "", nil, err
	}
	set, err := task.LoadTasks(root)
	if err != nil {
		return "", nil, err
	}
	return root, set, nil
}

// writeTaskAndIndex writes a mutated task and regenerates TASKS.md, both via the
// core store's atomic path. A transition and its derived index land in one commit
// when doCommit is set; doPush then shares it, treating a rejected push as
// "someone else got there first". It stays silent in JSON mode so the command can
// emit a clean payload. It sets t.Path to the written path.
func writeTaskAndIndex(root string, t *task.Task, commitMsg string, doCommit, doPush bool) error {
	store := core.NewStore(nil)

	op, err := task.Plan(root, t)
	if err != nil {
		return err
	}
	if !flagTaskJSON {
		fmt.Println(op.DiffText())
	}
	if !op.NoChange() {
		if err := store.Apply(op); err != nil {
			return err
		}
	}
	t.Path = op.AbsPath

	// The index is derived; regenerate it from disk after the task write.
	idxOp, err := task.PlanIndex(root)
	if err != nil {
		return err
	}
	if !idxOp.NoChange() {
		if err := store.Apply(idxOp); err != nil {
			return err
		}
	}

	var paths []string
	if !op.NoChange() {
		paths = append(paths, op.AbsPath)
	}
	if !idxOp.NoChange() {
		paths = append(paths, idxOp.AbsPath)
	}
	if doCommit && len(paths) > 0 {
		if !core.IsRepo(root) {
			return fmt.Errorf("wrote %d file(s) but --commit needs a git repo at %s", len(paths), root)
		}
		if err := core.Commit(root, commitMsg, paths...); err != nil {
			return err
		}
		if !flagTaskJSON {
			fmt.Println("committed:", commitMsg)
		}
	} else if len(paths) > 0 && !flagTaskJSON {
		fmt.Println("wrote (not committed; re-run with --commit or commit yourself):", op.AbsPath)
	}

	if doPush {
		if err := core.Push(root); err != nil {
			return fmt.Errorf("push rejected — another checkout may have claimed %s first; re-read and choose another task (%v)", t.ID, err)
		}
	}
	return nil
}

func emitJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// --- views -----------------------------------------------------------------

type progressView struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

type taskView struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Status   string       `json:"status"`
	Assignee string       `json:"assignee,omitempty"`
	Tags     []string     `json:"tags,omitempty"`
	Depends  []string     `json:"depends,omitempty"`
	Blockers []string     `json:"blockers,omitempty"`
	Progress progressView `json:"progress"`
	Path     string       `json:"path,omitempty"`
	Body     string       `json:"body,omitempty"`
}

func viewOf(set *task.TaskSet, t *task.Task) taskView {
	done, total, _ := t.Progress()
	return taskView{
		ID: t.ID, Title: t.Title, Status: string(t.Status), Assignee: t.Assignee,
		Tags: t.Tags, Depends: t.Depends, Blockers: set.Blockers(t.ID),
		Progress: progressView{Done: done, Total: total}, Path: t.Path,
	}
}

// --- new -------------------------------------------------------------------

func cmdTaskNew() *cobra.Command {
	var tags, depends []string
	c := &cobra.Command{
		Use:   "new <title>",
		Short: "Allocate the next id and scaffold a task file",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, set, err := loadTasks()
			if err != nil {
				return err
			}
			_, today := clock()
			id := task.NextID(set.Tasks)
			t, err := task.New(id, joinArgs(args), tags, depends, today)
			if err != nil {
				return err
			}
			if err := writeTaskAndIndex(root, t, "task new: "+id, flagCommit, false); err != nil {
				return err
			}
			if flagTaskJSON {
				return emitJSON(map[string]string{"id": id, "path": t.Path})
			}
			fmt.Println(id, t.Path)
			return nil
		},
	}
	c.Flags().StringArrayVar(&tags, "tag", nil, "tag (repeatable)")
	c.Flags().StringArrayVar(&depends, "depends", nil, "dependency task id (repeatable)")
	return c
}

// --- list ------------------------------------------------------------------

func cmdTaskList() *cobra.Command {
	var status, tag, assignee string
	var readyOnly bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List tasks, filtered; table or --json array",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, set, err := loadTasks()
			if err != nil {
				return err
			}
			var tasks []*task.Task
			if readyOnly {
				tasks, err = set.Ready()
				if err != nil {
					return err
				}
			} else {
				tasks = set.All()
			}
			var out []*task.Task
			for _, t := range tasks {
				if status != "" && string(t.Status) != status {
					continue
				}
				if assignee != "" && t.Assignee != assignee {
					continue
				}
				if tag != "" && !hasTag(t, tag) {
					continue
				}
				out = append(out, t)
			}
			if flagTaskJSON {
				views := make([]taskView, 0, len(out))
				for _, t := range out {
					views = append(views, viewOf(set, t))
				}
				return emitJSON(views)
			}
			printTaskTable(out)
			return nil
		},
	}
	c.Flags().StringVar(&status, "status", "", "filter by status")
	c.Flags().StringVar(&tag, "tag", "", "filter by tag")
	c.Flags().StringVar(&assignee, "assignee", "", "filter by assignee (agent:name or human:name)")
	c.Flags().BoolVar(&readyOnly, "ready", false, "only tasks that are ready to work")
	return c
}

func hasTag(t *task.Task, tag string) bool {
	return slices.Contains(t.Tags, tag)
}

func printTaskTable(tasks []*task.Task) {
	if len(tasks) == 0 {
		fmt.Println("(no tasks)")
		return
	}
	for _, t := range tasks {
		done, total, _ := t.Progress()
		assignee := t.Assignee
		if assignee == "" {
			assignee = "-"
		}
		fmt.Printf("%-8s %-8s %d/%-3d %-18s %s\n", t.ID, t.Status, done, total, assignee, t.Title)
	}
}

// --- show ------------------------------------------------------------------

func cmdTaskShow() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a full task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, set, err := loadTasks()
			if err != nil {
				return err
			}
			t := set.Get(args[0])
			if t == nil {
				return fmt.Errorf("no task %q", args[0])
			}
			if flagTaskJSON {
				v := viewOf(set, t)
				v.Body = string(t.Body())
				return emitJSON(v)
			}
			fmt.Print(string(t.Bytes()))
			return nil
		},
	}
}

// --- next ------------------------------------------------------------------

func cmdTaskNext() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Print the highest-priority ready task; non-zero exit if none",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, set, err := loadTasks()
			if err != nil {
				return err
			}
			t, err := set.Next()
			if err != nil {
				return err
			}
			if t == nil {
				return fmt.Errorf("no ready task")
			}
			if flagTaskJSON {
				return emitJSON(viewOf(set, t))
			}
			fmt.Println(t.ID, t.Title)
			return nil
		},
	}
}

// --- claim -----------------------------------------------------------------

func cmdTaskClaim() *cobra.Command {
	var as string
	var force, push bool
	c := &cobra.Command{
		Use:   "claim <id> --as agent:name",
		Short: "Claim a task: lock it, set status doing and assignee",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if as == "" {
				return fmt.Errorf("--as agent:name or human:name is required")
			}
			if !task.ValidAssignee(as) {
				return fmt.Errorf("malformed --as %q (want agent:name or human:name)", as)
			}
			root, set, err := loadTasks()
			if err != nil {
				return err
			}
			t := set.Get(args[0])
			if t == nil {
				return fmt.Errorf("no task %q", args[0])
			}
			if blockers := set.Blockers(t.ID); len(blockers) > 0 {
				return fmt.Errorf("cannot claim %s: dependencies not done: %s", t.ID, strings.Join(blockers, ", "))
			}

			// Advisory lock: take it before touching state; the loser reports the holder.
			_, broke, err := task.AcquireLock(root, t.ID, as, task.DefaultLockTTL, force)
			if err != nil {
				var le *task.ErrLocked
				if errors.As(err, &le) {
					return fmt.Errorf("cannot claim %s: %s (use --force to break a stale lock)", t.ID, le.Error())
				}
				return err
			}
			if broke != nil && !flagTaskJSON {
				fmt.Printf("warning: broke lock previously held by %s (since %s)\n", broke.Holder, broke.Since)
			}
			// Release the lock if the claim fails to land.
			done := false
			defer func() {
				if !done {
					_ = task.ReleaseLock(root, t.ID)
				}
			}()

			if err := task.TransitionTask(t, task.StatusDoing); err != nil {
				return err
			}
			if err := t.SetAssignee(as); err != nil {
				return err
			}
			_, today := clock()
			if err := t.Touch(today); err != nil {
				return err
			}
			// The claim commit writes the transition and its index; --push shares it.
			if err := writeTaskAndIndex(root, t, "task claim: "+t.ID+" by "+as, flagCommit || push, push); err != nil {
				return err
			}
			done = true
			if flagTaskJSON {
				return emitJSON(viewOf(set, t))
			}
			return nil
		},
	}
	c.Flags().StringVar(&as, "as", "", "claimant identity, agent:name or human:name (required)")
	c.Flags().BoolVar(&force, "force", false, "break a stale lock, warning about the previous holder")
	c.Flags().BoolVar(&push, "push", false, "commit the claim alone and push; a rejected push means reselect")
	return c
}

// --- release ---------------------------------------------------------------

func cmdTaskRelease() *cobra.Command {
	return &cobra.Command{
		Use:   "release <id>",
		Short: "Release a task: clear assignee and return to todo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, set, err := loadTasks()
			if err != nil {
				return err
			}
			t := set.Get(args[0])
			if t == nil {
				return fmt.Errorf("no task %q", args[0])
			}
			if err := task.TransitionTask(t, task.StatusTodo); err != nil {
				return err
			}
			if err := t.ClearAssignee(); err != nil {
				return err
			}
			_, today := clock()
			if err := t.Touch(today); err != nil {
				return err
			}
			if err := task.ReleaseLock(root, t.ID); err != nil {
				return err
			}
			return finishTask(root, set, t, "task release: "+t.ID)
		},
	}
}

// --- status ----------------------------------------------------------------

func cmdTaskStatus() *cobra.Command {
	var allowIncomplete bool
	c := &cobra.Command{
		Use:   "status <id> <status> [reason]",
		Short: "Transition a task, validated against the state machine",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, set, err := loadTasks()
			if err != nil {
				return err
			}
			t := set.Get(args[0])
			if t == nil {
				return fmt.Errorf("no task %q", args[0])
			}
			to := task.Status(args[1])
			reason := strings.TrimSpace(joinArgs(args[2:]))

			if to == task.StatusReview && !allowIncomplete {
				ok, err := task.ReviewComplete(t)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("cannot move %s to review: acceptance criteria incomplete (use --allow-incomplete to override)", t.ID)
				}
			}
			if to == task.StatusBlocked && reason == "" {
				return fmt.Errorf("blocked requires a reason: flow task status %s blocked \"why\"", t.ID)
			}

			if err := task.TransitionTask(t, to); err != nil {
				return err
			}
			_, today := clock()
			if to == task.StatusBlocked {
				if err := t.AppendLog(logEntry(today, identityOf(t), "blocked: "+reason)); err != nil {
					return err
				}
			}
			if err := t.Touch(today); err != nil {
				return err
			}
			return finishTask(root, set, t, "task status: "+t.ID+" -> "+string(to))
		},
	}
	c.Flags().BoolVar(&allowIncomplete, "allow-incomplete", false, "permit review with unticked criteria")
	return c
}

// --- check / uncheck -------------------------------------------------------

func cmdTaskCheck() *cobra.Command {
	return checkCommand("check", "Tick an acceptance criterion by 1-based index", true)
}

func cmdTaskUncheck() *cobra.Command {
	return checkCommand("uncheck", "Untick an acceptance criterion by 1-based index", false)
}

func checkCommand(use, short string, checked bool) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <id> <n>",
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, set, err := loadTasks()
			if err != nil {
				return err
			}
			t := set.Get(args[0])
			if t == nil {
				return fmt.Errorf("no task %q", args[0])
			}
			n, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("index must be a number: %v", err)
			}
			if checked {
				err = t.Check(n)
			} else {
				err = t.Uncheck(n)
			}
			if err != nil {
				return err
			}
			_, today := clock()
			if err := t.Touch(today); err != nil {
				return err
			}
			return finishTask(root, set, t, fmt.Sprintf("task %s: %s #%d", use, t.ID, n))
		},
	}
}

// --- log -------------------------------------------------------------------

func cmdTaskLog() *cobra.Command {
	return &cobra.Command{
		Use:   "log <id> <message>",
		Short: "Append a dated line to the Log section",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, set, err := loadTasks()
			if err != nil {
				return err
			}
			t := set.Get(args[0])
			if t == nil {
				return fmt.Errorf("no task %q", args[0])
			}
			_, today := clock()
			if err := t.AppendLog(logEntry(today, identityOf(t), joinArgs(args[1:]))); err != nil {
				return err
			}
			if err := t.Touch(today); err != nil {
				return err
			}
			return finishTask(root, set, t, "task log: "+t.ID)
		},
	}
}

// --- sync ------------------------------------------------------------------

func cmdTaskSync() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Regenerate .flow/TASKS.md from the task files",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := mustRoot()
			if err != nil {
				return err
			}
			op, err := task.PlanIndex(root)
			if err != nil {
				return err
			}
			if !flagTaskJSON {
				fmt.Println(op.DiffText())
			}
			if !op.NoChange() {
				if err := core.NewStore(nil).Apply(op); err != nil {
					return err
				}
				if flagCommit {
					if !core.IsRepo(root) {
						return fmt.Errorf("wrote %s but --commit needs a git repo at %s", op.AbsPath, root)
					}
					if err := core.Commit(root, "task sync", op.AbsPath); err != nil {
						return err
					}
				}
			}
			if flagTaskJSON {
				return emitJSON(map[string]string{"path": op.AbsPath})
			}
			return nil
		},
	}
}

// --- shared mutation tail ---------------------------------------------------

// finishTask writes the mutated task, regenerates the index, and emits the task
// view in JSON mode.
func finishTask(root string, set *task.TaskSet, t *task.Task, commitMsg string) error {
	if err := writeTaskAndIndex(root, t, commitMsg, flagCommit, false); err != nil {
		return err
	}
	if flagTaskJSON {
		return emitJSON(viewOf(set, t))
	}
	return nil
}

// taskAgentsDoc is the agent protocol scaffolded at .flow/tasks/AGENTS.md. It is
// the contract an agent working the board must follow: short and imperative.
const taskAgentsDoc = `# Agent protocol for tasks

You are working the task board in ` + "`.flow/tasks/`" + `. Follow these rules.

1. Claim before working. Never edit a task you do not hold.
2. Append to the Log, never rewrite it. One line, dated, prefixed with your identity.
3. Tick an acceptance criterion only when a test proves it, not when the code is written.
4. Never set status to ` + "`done`" + `. Move to ` + "`review`" + ` and stop.
5. Release the task if you stop before finishing, and log why.
6. Do not edit ` + "`TASKS.md`" + `. It is generated.
`

// scaffoldTaskAgents writes the agent protocol at .flow/tasks/AGENTS.md, leaving an
// existing one untouched.
func scaffoldTaskAgents(root string) error {
	path := filepath.Join(root, ".flow", "tasks", "AGENTS.md")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(taskAgentsDoc), 0o644)
}

// ensureGitattributes adds `.flow/TASKS.md -diff` to the project's .gitattributes
// so the generated index stops cluttering reviews, without disturbing other rules.
func ensureGitattributes(root string) error {
	const line = ".flow/TASKS.md -diff"
	path := filepath.Join(root, ".gitattributes")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if slices.Contains(strings.Split(string(existing), "\n"), line) {
		return nil
	}
	out := string(existing)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += line + "\n"
	return os.WriteFile(path, []byte(out), 0o644)
}

// logEntry formats a Log line body (the bullet is added by AppendLog):
// "<date> <identity>: <message>".
func logEntry(date, identity, message string) string {
	return date + " " + identity + ": " + message
}

// identityOf derives a short log identity from the assignee (the name after the
// agent:/human: prefix), or "flow" when unassigned.
func identityOf(t *task.Task) string {
	if t.Assignee == "" {
		return "flow"
	}
	if i := strings.IndexByte(t.Assignee, ':'); i >= 0 {
		return t.Assignee[i+1:]
	}
	return t.Assignee
}
