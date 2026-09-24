package cli

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/sting8k/just-harness/internal/domain"
	"github.com/sting8k/just-harness/internal/store"
)

func storyAdd(e *env, args []string) error {
	f := newFlags("story add")
	title, lane := f.String("title"), f.String("lane")
	verify, contract := f.String("verify"), f.String("contract")
	noPacket := f.Bool("no-packet")
	if err := f.parse(args, "title", "lane"); err != nil {
		return err
	}
	if err := domain.CheckEnum("lane", *lane, domain.Lanes); err != nil {
		return inputf("%v", err)
	}
	if err := e.checkRepoPath("contract", *contract); err != nil {
		return err
	}
	id, err := e.st.NewID(domain.KindStory)
	if err != nil {
		return err
	}
	now := store.Now()
	s := domain.Story{V: domain.SchemaVersion, ID: id, Title: *title, Lane: *lane, Status: "planned",
		Contract: *contract, Verify: *verify, CreatedAt: now, UpdatedAt: now}
	if !*noPacket {
		s.Packet = store.ProsePath("stories", id, *title)
		if err := e.st.CreateProse(s.Packet, "story.md", id, *title); err != nil {
			return err
		}
	}
	if err := e.st.Create(domain.KindStory, id, &s); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "created %s\n", id)
	if s.Packet != "" {
		fmt.Fprintf(e.stdout, "packet  %s\n", s.Packet)
	}
	return nil
}

func storyUpdate(e *env, args []string) error {
	f := newFlags("story update")
	id := f.String("id")
	for _, n := range []string{"status", "title", "lane", "verify", "contract", "waive"} {
		f.String(n)
	}
	if err := f.parse(args, "id"); err != nil {
		return err
	}
	c := domain.StoryChange{Status: f.opt("status"), Title: f.opt("title"), Lane: f.opt("lane"),
		Verify: f.opt("verify"), Contract: f.opt("contract"), Waive: f.opt("waive")}
	if c == (domain.StoryChange{}) {
		return inputf("nothing to update")
	}
	if c.Contract != nil {
		if err := e.checkRepoPath("contract", *c.Contract); err != nil {
			return err
		}
	}
	var s domain.Story
	if err := e.load(domain.KindStory, *id, &s); err != nil {
		return err
	}
	// ApplyStoryChange mutates s; on any error nothing is written.
	gateErr, inErr := domain.ApplyStoryChange(&s, c, store.Now())
	if inErr != nil {
		return inputf("%v", inErr)
	}
	if gateErr != nil {
		return fmt.Errorf("gate refused: %v", gateErr)
	}
	if err := e.st.Save(domain.KindStory, s.ID, &s); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "updated %s [%s] %s%s\n", s.ID, s.Lane, s.Status, waiverNote(&s))
	return nil
}

func waiverNote(s *domain.Story) string {
	if s.Waiver == "" {
		return ""
	}
	return fmt.Sprintf(" — WAIVED: %q", s.Waiver)
}

// timeoutExitCode mirrors coreutils `timeout`.
const timeoutExitCode = 124

func storyVerify(e *env, args []string) error {
	f := newFlags("story verify")
	id, timeout := f.String("id"), f.String("timeout")
	if err := f.parse(args, "id"); err != nil {
		return err
	}
	var limit time.Duration
	if *timeout != "" {
		d, err := time.ParseDuration(*timeout)
		if err != nil || d <= 0 {
			return inputf("--timeout: invalid duration %q (e.g. 90s, 10m)", *timeout)
		}
		limit = d
	}
	var s domain.Story
	if err := e.load(domain.KindStory, *id, &s); err != nil {
		return err
	}
	if s.Verify == "" {
		return inputf("%s has no verify command; set one with `story update --id %s --verify \"cmd\"`", s.ID, s.ID)
	}

	ctx := context.Background()
	if limit > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, limit)
		defer cancel()
	}
	cmd := shellCommand(ctx, s.Verify)
	cmd.Dir = e.st.Root
	cmd.Stdout, cmd.Stderr = e.stdout, e.stderr
	cmd.WaitDelay = 2 * time.Second // don't hang on grandchildren holding the pipes
	fmt.Fprintf(e.stderr, "$ %s\n", s.Verify)
	runErr := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		code = timeoutExitCode
		fmt.Fprintf(e.stderr, "verify timed out after %s\n", limit)
	case errors.As(runErr, &exitErr):
		code = exitErr.ExitCode()
	case runErr != nil:
		return fmt.Errorf("could not run verify: %w", runErr)
	}

	lv := &domain.Verify{Result: "pass", ExitCode: code, Command: s.Verify, At: store.Now()}
	if code != 0 {
		lv.Result = "fail"
	}
	lv.Commit, lv.Dirty = gitState(e.st.Root)
	s.LastVerify, s.UpdatedAt = lv, lv.At
	if err := e.st.Save(domain.KindStory, s.ID, &s); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "verify %s: %s (exit %d)\n", s.ID, lv.Result, code)
	if code != 0 {
		return exitCode(code)
	}
	return nil
}

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "sh", "-c", command)
}

// gitState returns HEAD's short sha and whether the tree has uncommitted
// changes outside .harness/ (records written by the CLI don't count).
// Outside a git repo both are empty.
func gitState(root string) (commit string, dirty bool) {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", false
	}
	commit = strings.TrimSpace(string(out))
	status, err := exec.Command("git", "-C", root, "status", "--porcelain", "--", ".", ":(exclude)"+store.Dir).Output()
	return commit, err == nil && len(strings.TrimSpace(string(status))) > 0
}

func headCommit(root string) string {
	c, _ := gitState(root)
	return c
}

func decisionAdd(e *env, args []string) error {
	f := newFlags("decision add")
	title, status, storyID := f.String("title"), f.String("status"), f.String("story")
	if err := f.parse(args, "title"); err != nil {
		return err
	}
	if *status == "" {
		*status = "proposed"
	}
	if *status != "proposed" && *status != "accepted" {
		return inputf("--status: new decisions are proposed|accepted, got %q", *status)
	}
	if err := e.requireStory(*storyID); err != nil {
		return err
	}
	id, err := e.st.NewID(domain.KindDecision)
	if err != nil {
		return err
	}
	now := store.Now()
	d := domain.Decision{V: domain.SchemaVersion, ID: id, Title: *title, Status: *status,
		Doc: store.ProsePath("decisions", id, *title), Story: *storyID, CreatedAt: now, UpdatedAt: now}
	if err := e.st.CreateProse(d.Doc, "decision.md", id, *title); err != nil {
		return err
	}
	if err := e.st.Create(domain.KindDecision, id, &d); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "created %s\ndoc     %s\n", id, d.Doc)
	return nil
}

func decisionUpdate(e *env, args []string) error {
	f := newFlags("decision update")
	id, status, by := f.String("id"), f.String("status"), f.String("by")
	if err := f.parse(args, "id", "status"); err != nil {
		return err
	}
	if err := domain.CheckEnum("status", *status, domain.DecisionStatuses); err != nil {
		return inputf("%v", err)
	}
	if *by != "" {
		if *status != "superseded" {
			return inputf("--by only applies with --status superseded")
		}
		if *by == *id {
			return inputf("a decision cannot supersede itself")
		}
		if err := e.requireRecord(domain.KindDecision, *by); err != nil {
			return err
		}
	}
	var d domain.Decision
	if err := e.load(domain.KindDecision, *id, &d); err != nil {
		return err
	}
	d.Status, d.SupersededBy, d.UpdatedAt = *status, *by, store.Now()
	if err := e.st.Save(domain.KindDecision, d.ID, &d); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "updated %s %s\n", d.ID, d.Status)
	return nil
}

func guardrailAdd(e *env, args []string) error {
	f := newFlags("guardrail add")
	rule, why := f.String("rule"), f.String("why")
	if err := f.parse(args, "rule", "why"); err != nil {
		return err
	}
	id, err := e.st.NewID(domain.KindGuardrail)
	if err != nil {
		return err
	}
	now := store.Now()
	g := domain.Guardrail{V: domain.SchemaVersion, ID: id, Rule: *rule, Why: *why, Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := e.st.Create(domain.KindGuardrail, id, &g); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "created %s\n", id)
	return nil
}

func guardrailSupersede(e *env, args []string) error {
	f := newFlags("guardrail supersede")
	id := f.String("id")
	if err := f.parse(args, "id"); err != nil {
		return err
	}
	var g domain.Guardrail
	if err := e.load(domain.KindGuardrail, *id, &g); err != nil {
		return err
	}
	g.Status, g.UpdatedAt = "superseded", store.Now()
	if err := e.st.Save(domain.KindGuardrail, g.ID, &g); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "updated %s superseded\n", g.ID)
	return nil
}

func cmdTrace(e *env, args []string) error {
	f := newFlags("trace")
	summary, outcome, storyID := f.String("summary"), f.String("outcome"), f.String("story")
	evidence, unverified, agent := f.String("evidence"), f.String("unverified"), f.String("agent")
	if err := f.parse(args, "summary", "outcome"); err != nil {
		return err
	}
	if err := domain.CheckEnum("outcome", *outcome, domain.TraceOutcomes); err != nil {
		return inputf("%v", err)
	}
	if err := e.requireStory(*storyID); err != nil {
		return err
	}
	id, err := e.st.NewID(domain.KindTrace)
	if err != nil {
		return err
	}
	t := domain.Trace{V: domain.SchemaVersion, ID: id, Summary: *summary, Outcome: *outcome, Story: *storyID,
		Evidence: *evidence, Unverified: *unverified, Agent: *agent, Commit: headCommit(e.st.Root), CreatedAt: store.Now()}
	if err := e.st.Create(domain.KindTrace, id, &t); err != nil {
		return err
	}
	fmt.Fprintf(e.stdout, "created %s\n", id)
	return nil
}

func (e *env) requireStory(id string) error {
	if id == "" {
		return nil
	}
	return e.requireRecord(domain.KindStory, id)
}

func (e *env) requireRecord(k domain.Kind, id string) error {
	if err := domain.CheckID(k, id); err != nil {
		return inputf("%v", err)
	}
	if !e.st.Exists(k, id) {
		return inputf("%s does not exist", id)
	}
	return nil
}

// load reads an existing record; a malformed or unknown id is an input error.
func (e *env) load(k domain.Kind, id string, v any) error {
	if err := e.requireRecord(k, id); err != nil {
		return err
	}
	return e.st.Load(k, id, v)
}
