package cli

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/sting8k/just-harness/internal/domain"
)

func cmdQuery(e *env, args []string) error {
	if len(args) == 0 {
		return inputf("query needs a view: status|stories|decisions|guardrails|traces")
	}
	view, rest := args[0], args[1:]
	if view == "status" {
		if err := newFlags("query status").parse(rest); err != nil {
			return err
		}
		snap, err := load(e.st)
		if err != nil {
			return err
		}
		printStatus(e.stdout, snap)
		return nil
	}

	f := newFlags("query " + view)
	status, md := f.String("status"), f.Bool("md")
	if err := f.parse(rest); err != nil {
		return err
	}
	enums := map[string][]string{"stories": domain.StoryStatuses, "decisions": domain.DecisionStatuses,
		"guardrails": domain.GuardrailStatuses, "traces": domain.TraceOutcomes}
	allowed, ok := enums[view]
	if !ok {
		return inputf("unknown query view %q (status|stories|decisions|guardrails|traces)", view)
	}
	if *status != "" {
		if err := domain.CheckEnum("--status", *status, allowed); err != nil {
			return inputf("%v", err)
		}
	}
	snap, err := load(e.st)
	if err != nil {
		return err
	}
	keep := func(s string) bool { return *status == "" || s == *status }
	var header []string
	var rows [][]string
	switch view {
	case "stories":
		header = []string{"ID", "LANE", "STATUS", "PROOF", "TITLE", "PACKET"}
		for _, s := range snap.stories {
			if keep(s.Status) {
				rows = append(rows, []string{s.ID, s.Lane, s.Status, proof(&s), s.Title, s.Packet})
			}
		}
	case "decisions":
		header = []string{"ID", "STATUS", "TITLE", "STORY", "SUPERSEDED_BY", "DOC"}
		for _, d := range snap.decisions {
			if keep(d.Status) {
				rows = append(rows, []string{d.ID, d.Status, d.Title, d.Story, d.SupersededBy, d.Doc})
			}
		}
	case "guardrails":
		header = []string{"ID", "STATUS", "RULE", "WHY"}
		for _, g := range snap.guardrails {
			if keep(g.Status) {
				rows = append(rows, []string{g.ID, g.Status, g.Rule, g.Why})
			}
		}
	case "traces":
		header = []string{"ID", "CREATED", "OUTCOME", "STORY", "SUMMARY", "UNVERIFIED"}
		for _, t := range snap.traces {
			if keep(t.Outcome) {
				rows = append(rows, []string{t.ID, t.CreatedAt, t.Outcome, t.Story, t.Summary, t.Unverified})
			}
		}
	}
	if *md {
		renderMarkdown(e.stdout, header, rows)
	} else {
		renderText(e.stdout, header, rows)
	}
	if n := len(snap.problems); n > 0 {
		fmt.Fprintf(e.stderr, "warning: %d problem(s) in records; run `check`\n", n)
	}
	return nil
}

// proof summarizes a story's evidence; a waiver is always shown apart from a pass (I4).
func proof(s *domain.Story) string {
	var parts []string
	if lv := s.LastVerify; lv != nil {
		p := lv.Result
		if lv.Command != s.Verify {
			p += " (stale)"
		} else if lv.Dirty {
			p += " (dirty)"
		}
		parts = append(parts, p)
	}
	if s.Waiver != "" {
		parts = append(parts, fmt.Sprintf("WAIVED: %q", s.Waiver))
	}
	return strings.Join(parts, ", ")
}

func printStatus(w io.Writer, snap *snapshot) {
	section := func(label string, lines []string) {
		if len(lines) == 0 {
			lines = []string{"none"}
		}
		for i, l := range lines {
			if i > 0 {
				label = ""
			}
			fmt.Fprintf(w, "%-21s%s\n", label, l)
		}
	}
	var guard, open, attention, proposed []string
	for _, g := range snap.guardrails {
		if g.Status == "active" {
			guard = append(guard, fmt.Sprintf("%s %s", g.ID, oneLine(g.Rule)))
		}
	}
	for _, s := range snap.stories {
		switch s.Status {
		case "planned", "in_progress", "changed":
			line := fmt.Sprintf("%s [%s] %s — %s", s.ID, s.Lane, s.Status, oneLine(s.Title))
			if s.Packet != "" {
				line += "   (" + s.Packet + ")"
			}
			open = append(open, line)
		case "implemented":
			if issue := s.GateIssue(); issue != "" {
				attention = append(attention, fmt.Sprintf("%s [%s] implemented — %s", s.ID, s.Lane, issue))
			} else if s.Waiver != "" {
				attention = append(attention, fmt.Sprintf("%s [%s] implemented — WAIVED: %q", s.ID, s.Lane, oneLine(s.Waiver)))
			}
		}
	}
	for _, d := range snap.decisions {
		if d.Status == "proposed" {
			proposed = append(proposed, fmt.Sprintf("%s %s", d.ID, oneLine(d.Title)))
		}
	}
	section("Guardrails (active):", guard)
	section("Open stories:", open)
	section("Needs attention:", attention)
	section("Proposed decisions:", proposed)
	section("Problems (check):", snap.problems)
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func renderText(w io.Writer, header []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "(none)")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, r := range rows {
		cells := make([]string, len(r))
		for i, c := range r {
			if cells[i] = oneLine(c); cells[i] == "" {
				cells[i] = "-"
			}
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	tw.Flush()
}

// renderMarkdown writes a GFM table; | is escaped and newlines become <br>.
func renderMarkdown(w io.Writer, header []string, rows [][]string) {
	esc := strings.NewReplacer("|", `\|`, "\r\n", "<br>", "\n", "<br>")
	fmt.Fprintf(w, "| %s |\n", strings.Join(header, " | "))
	fmt.Fprintf(w, "|%s\n", strings.Repeat(" --- |", len(header)))
	for _, r := range rows {
		cells := make([]string, len(r))
		for i, c := range r {
			cells[i] = esc.Replace(c)
		}
		fmt.Fprintf(w, "| %s |\n", strings.Join(cells, " | "))
	}
}
