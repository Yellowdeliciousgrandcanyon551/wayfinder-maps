package wayfinder

import (
	"strings"
	"testing"
	"time"
)

func TestParseTicketFrontmatter(t *testing.T) {
	src := `---
type: grilling
status: claimed
blocked_by: [02, 03]
undermined_by: [08]
claimed_by: session-a
claimed_at: 2026-07-10T09:00:00Z
assets: [../assets/x.html]
---

# Re-shape DailyGroups

## Question
Body.
`
	tk, err := ParseTicket("p", "04-date-indexed.md", src)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Num != 4 || tk.Slug != "date-indexed" {
		t.Errorf("num/slug = %d/%q", tk.Num, tk.Slug)
	}
	if tk.Title != "Re-shape DailyGroups" {
		t.Errorf("title = %q", tk.Title)
	}
	if tk.Type != TypeGrilling || tk.Status != StatusClaimed {
		t.Errorf("type/status = %q/%q", tk.Type, tk.Status)
	}
	if len(tk.BlockedBy) != 2 || tk.BlockedBy[0] != 2 || tk.BlockedBy[1] != 3 {
		t.Errorf("blocked_by = %v", tk.BlockedBy)
	}
	if len(tk.UnderminedBy) != 1 || tk.UnderminedBy[0] != 8 {
		t.Errorf("undermined_by = %v", tk.UnderminedBy)
	}
	if tk.ClaimedBy != "session-a" || tk.ClaimedAt != "2026-07-10T09:00:00Z" {
		t.Errorf("claim = %q/%q", tk.ClaimedBy, tk.ClaimedAt)
	}
	if len(tk.Assets) != 1 {
		t.Errorf("assets = %v", tk.Assets)
	}
	if tk.Legacy {
		t.Error("frontmatter ticket flagged legacy")
	}
	if tk.HasAnswer {
		t.Error("no Answer section, but HasAnswer")
	}
}

func TestParseTicketLegacy(t *testing.T) {
	src := `# Muted empty-day card design

Type: prototype
Status: resolved
Blocked by: none

## Question
Q.

## Answer
A.
`
	tk, err := ParseTicket("p", "01-muted-empty-day-design.md", src)
	if err != nil {
		t.Fatal(err)
	}
	if !tk.Legacy {
		t.Error("loose header not flagged legacy")
	}
	if tk.Type != TypePrototype || tk.Status != StatusResolved {
		t.Errorf("type/status = %q/%q", tk.Type, tk.Status)
	}
	if len(tk.BlockedBy) != 0 {
		t.Errorf("blocked_by none = %v", tk.BlockedBy)
	}
	if !tk.HasAnswer {
		t.Error("Answer section missed")
	}
	if tk.Title != "Muted empty-day card design" {
		t.Errorf("title = %q", tk.Title)
	}
}

func TestParseMapSectionsAreScoped(t *testing.T) {
	// Notes links a ticket too; only Decisions-so-far links must be collected.
	src := `# The Map

## Destination
A spec.

## Notes
Superseded by [Window size](./tickets/02-window.md).

## Decisions so far

- [Muted empty day](./tickets/01-muted.md) — a ledger line.
- [Window size](./tickets/02-window.md) — 30 days.

## Not yet specified

- **Today's row.** Whether today is distinguished. <clears-with: 04>
- **Delete's new home.** Unexamined.

## Out of scope

- [Collapsing runs](./tickets/03-collapse.md) — user ruled it out.
`
	m := ParseMap("map.md", src)
	if m.Name != "The Map" {
		t.Errorf("name = %q", m.Name)
	}
	if len(m.Decisions) != 2 {
		t.Fatalf("decisions = %d, want 2 (Notes link must not leak in)", len(m.Decisions))
	}
	if m.Decisions[0].TicketNum != 1 || m.Decisions[1].TicketNum != 2 {
		t.Errorf("decisions = %v", m.Decisions)
	}
	if len(m.OutOfScope) != 1 || m.OutOfScope[0].TicketNum != 3 {
		t.Errorf("out of scope = %v", m.OutOfScope)
	}
	if len(m.Fog) != 2 {
		t.Fatalf("fog = %d, want 2", len(m.Fog))
	}
	if m.Fog[0].Title != "Today's row" || m.Fog[0].ClearsWith != 4 {
		t.Errorf("fog[0] = %+v", m.Fog[0])
	}
	if m.Fog[1].ClearsWith != 0 {
		t.Errorf("unanchored fog got anchor %d", m.Fog[1].ClearsWith)
	}
	if m.Decisions[0].Line == 0 {
		t.Error("decision line not recorded")
	}
}

// effort builds an in-memory effort; mapSrc is parsed so map-side checks run.
func effort(mapSrc string, ts ...*Ticket) *Effort {
	return &Effort{Dir: "d", Name: "e", Map: ParseMap("map.md", mapSrc), Tickets: ts}
}

func tk(num int, status Status, blocked ...int) *Ticket {
	t := &Ticket{
		Num: num, Path: "t.md", Title: "T" + string(rune('0'+num)),
		Type: TypeGrilling, Status: status, BlockedBy: blocked,
	}
	t.HasAnswer = status.Closed()
	return t
}

const okMap = "# M\n\n## Destination\nD.\n\n## Decisions so far\n\n## Not yet specified\n\n## Out of scope\n"

func diagsContain(d []Diagnostic, sub string) bool {
	for _, x := range d {
		if strings.Contains(x.Msg, sub) {
			return true
		}
	}
	return false
}

func TestFrontier(t *testing.T) {
	e := effort(okMap+"\n", tk(1, StatusResolved), tk(2, StatusOpen), tk(3, StatusOpen, 1), tk(4, StatusOpen, 2), tk(5, StatusClaimed))
	got := e.Frontier()
	if len(got) != 2 || got[0].Num != 2 || got[1].Num != 3 {
		t.Fatalf("frontier = %v, want [02 03]", got)
	}
}

func TestFrontierExcludesOutOfScopeBlocker(t *testing.T) {
	e := effort(okMap, tk(1, StatusOutOfScope), tk(2, StatusOpen, 1))
	if len(e.Frontier()) != 0 {
		t.Error("out-of-scope blocker must not unblock a dependent")
	}
}

func TestLintCatchesDrift(t *testing.T) {
	tests := []struct {
		name   string
		effort *Effort
		want   string
	}{
		{
			"resolved missing from decisions",
			effort(okMap, tk(1, StatusResolved)),
			"absent from the map's Decisions-so-far",
		},
		{
			"resolved without answer",
			func() *Effort {
				t1 := tk(1, StatusResolved)
				t1.HasAnswer = false
				return effort("# M\n\n## Destination\nD.\n\n## Decisions so far\n\n- [T](./tickets/01-a.md) — g.\n", t1)
			}(),
			"no `## Answer` section",
		},
		{
			"blocked_by a ticket that does not exist",
			effort(okMap, tk(1, StatusOpen, 42)),
			"blocked_by 42, which does not exist",
		},
		{
			"cycle",
			effort(okMap, tk(1, StatusOpen, 2), tk(2, StatusOpen, 1)),
			"blocked_by cycle",
		},
		{
			"self block",
			effort(okMap, tk(1, StatusOpen, 1)),
			"blocked_by itself",
		},
		{
			"out of scope listed as a decision",
			func() *Effort {
				t1 := tk(1, StatusOutOfScope)
				return effort("# M\n\n## Destination\nD.\n\n## Decisions so far\n\n- [T](./tickets/01-a.md) — g.\n\n## Out of scope\n\n- [T](./tickets/01-a.md) — ruled out.\n", t1)
			}(),
			"a scope boundary is not a step on the route",
		},
		{
			"out of scope missing from Out-of-scope section",
			effort(okMap, tk(1, StatusOutOfScope)),
			"absent from the map's Out-of-scope section",
		},
		{
			"decisions links an open ticket",
			effort("# M\n\n## Destination\nD.\n\n## Decisions so far\n\n- [T](./tickets/01-a.md) — g.\n", tk(1, StatusOpen)),
			"which is open, not resolved",
		},
		{
			"fog duplicates a live ticket",
			func() *Effort {
				t1 := tk(4, StatusOpen)
				t1.Title = "Today's row"
				return effort("# M\n\n## Destination\nD.\n\n## Not yet specified\n\n- **Today's row.** Prose.\n", t1)
			}(),
			"duplicates live ticket 04",
		},
		{
			"fog anchored to an already-resolved ticket",
			func() *Effort {
				src := "# M\n\n## Destination\nD.\n\n## Decisions so far\n\n- [T](./tickets/01-a.md) — g.\n\n## Not yet specified\n\n- **Some fog.** Prose. <clears-with: 01>\n"
				return effort(src, tk(1, StatusResolved))
			}(),
			"should have graduated",
		},
		{
			"blocked by an out-of-scope ticket",
			func() *Effort {
				src := "# M\n\n## Destination\nD.\n\n## Out of scope\n\n- [T](./tickets/01-a.md) — ruled out.\n"
				return effort(src, tk(1, StatusOutOfScope), tk(2, StatusOpen, 1))
			}(),
			"can never unblock",
		},
		{
			"no destination",
			effort("# M\n\n## Notes\nn.\n"),
			"no Destination",
		},
		{
			"bad status",
			effort(okMap, &Ticket{Num: 1, Path: "t.md", Title: "T", Type: TypeGrilling, Status: "done"}),
			`status "done" is not`,
		},
		{
			"bad type",
			effort(okMap, &Ticket{Num: 1, Path: "t.md", Title: "T", Type: "chore", Status: StatusOpen}),
			`type "chore" is not`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := Lint(tc.effort, DefaultOptions())
			if !diagsContain(d, tc.want) {
				t.Errorf("no diagnostic containing %q; got %v", tc.want, d)
			}
		})
	}
}

func TestLintStaleClaim(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	t1 := tk(1, StatusClaimed)
	t1.ClaimedBy = "session-a"
	t1.ClaimedAt = now.Add(-100 * time.Hour).Format(time.RFC3339)

	opt := Options{StaleClaimAfter: 72 * time.Hour, Now: now}
	if d := Lint(effort(okMap, t1), opt); !diagsContain(d, "likely a dead session") {
		t.Errorf("stale claim not caught: %v", d)
	}

	t1.ClaimedAt = now.Add(-1 * time.Hour).Format(time.RFC3339)
	if d := Lint(effort(okMap, t1), opt); diagsContain(d, "likely a dead session") {
		t.Error("fresh claim flagged stale")
	}
}

func TestLintDuplicateTicketNumber(t *testing.T) {
	a, b := tk(10, StatusOpen), tk(10, StatusOpen)
	b.Path = "other.md"
	if d := Lint(effort(okMap, a, b), DefaultOptions()); !diagsContain(d, "duplicate ticket number 10") {
		t.Errorf("collision not caught: %v", d)
	}
}

func TestLintCleanMapIsSilent(t *testing.T) {
	src := "# M\n\n## Destination\nD.\n\n## Decisions so far\n\n- [One](./tickets/01-a.md) — gist.\n\n## Not yet specified\n\n- **Open fog.** prose.\n\n## Out of scope\n"
	t1 := tk(1, StatusResolved)
	t2 := tk(2, StatusOpen, 1)
	if d := Lint(effort(src, t1, t2), DefaultOptions()); len(d) != 0 {
		t.Errorf("clean map produced %v", d)
	}
}
