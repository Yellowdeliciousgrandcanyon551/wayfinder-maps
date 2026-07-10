package wayfinder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusClaimed    Status = "claimed"
	StatusResolved   Status = "resolved"
	StatusOutOfScope Status = "out_of_scope"
)

// Closed reports whether the ticket is off the frontier for good. Out-of-scope
// tickets are closed but are not decisions on the route, so they never satisfy
// a Blocked-by edge.
func (s Status) Closed() bool { return s == StatusResolved || s == StatusOutOfScope }

type Type string

const (
	TypeResearch  Type = "research"
	TypePrototype Type = "prototype"
	TypeGrilling  Type = "grilling"
	TypeTask      Type = "task"
)

type Ticket struct {
	Num          int
	Slug         string
	Path         string
	Title        string
	Type         Type
	Status       Status
	BlockedBy    []int
	UnderminedBy []int
	ClaimedBy    string
	ClaimedAt    string
	Assets       []string

	HasAnswer bool
	Legacy    bool // loose "Type:" header rather than YAML frontmatter
}

type FogPatch struct {
	Title      string
	ClearsWith int // 0 when unanchored
	Line       int
}

type Decision struct {
	TicketNum int
	Line      int
}

type Map struct {
	Path        string
	Name        string
	Destination string
	Fog         []FogPatch
	Decisions   []Decision
	OutOfScope  []Decision
}

type Effort struct {
	Dir     string
	Name    string
	Map     *Map
	Tickets []*Ticket
}

func (e *Effort) ByNum(n int) *Ticket {
	for _, t := range e.Tickets {
		if t.Num == n {
			return t
		}
	}
	return nil
}

// Frontier returns the open, unclaimed tickets whose every blocker is resolved,
// in ticket-number order — the edge of the known.
func (e *Effort) Frontier() []*Ticket {
	var out []*Ticket
	for _, t := range e.Tickets {
		if t.Status != StatusOpen {
			continue
		}
		ready := true
		for _, b := range t.BlockedBy {
			dep := e.ByNum(b)
			if dep == nil || dep.Status != StatusResolved {
				ready = false
				break
			}
		}
		if ready {
			out = append(out, t)
		}
	}
	return out
}

func (e *Effort) Count(s Status) int {
	n := 0
	for _, t := range e.Tickets {
		if t.Status == s {
			n++
		}
	}
	return n
}

var (
	reH1        = regexp.MustCompile(`(?m)^# (.+?)\s*$`)
	reAnswer    = regexp.MustCompile(`(?m)^## Answer\s*$`)
	reLegacyKey = regexp.MustCompile(`(?m)^(Type|Status|Blocked by):[ \t]*(.*)$`)
	reFilename  = regexp.MustCompile(`^(\d+)-(.+)\.md$`)
	reDecision  = regexp.MustCompile(`\(\./tickets/(\d+)-[^)]*\)`)
	reFogTitle  = regexp.MustCompile(`^-\s+\*\*(.+?)\*\*`)
	reClearsRaw = regexp.MustCompile(`clears-with:\s*(\d+)`)
)

// parseFrontmatter splits a leading `---` delimited block into key/value pairs.
// Values are raw strings; list values keep their brackets for splitList.
func parseFrontmatter(src string) (map[string]string, string, bool) {
	if !strings.HasPrefix(src, "---\n") {
		return nil, src, false
	}
	end := strings.Index(src[4:], "\n---")
	if end < 0 {
		return nil, src, false
	}
	block := src[4 : 4+end]
	rest := src[4+end:]
	if i := strings.Index(rest, "\n"); i >= 0 {
		rest = rest[i+1:]
		if j := strings.Index(rest, "\n"); j >= 0 {
			rest = rest[j+1:]
		}
	}

	kv := map[string]string{}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i < 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if c := strings.Index(val, " #"); c >= 0 {
			val = strings.TrimSpace(val[:c])
		}
		kv[key] = val
	}
	return kv, rest, true
}

func splitList(v string) []string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "[")
	v = strings.TrimSuffix(v, "]")
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "none") {
		return nil
	}
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitNums(v string) ([]int, error) {
	var out []int
	for _, p := range splitList(v) {
		n, err := strconv.Atoi(strings.TrimLeft(p, "0"))
		if err != nil {
			return nil, fmt.Errorf("%q is not a ticket number", p)
		}
		out = append(out, n)
	}
	return out, nil
}

func ParseTicket(path, filename, src string) (*Ticket, error) {
	m := reFilename.FindStringSubmatch(filename)
	if m == nil {
		return nil, fmt.Errorf("filename %q is not NN-slug.md", filename)
	}
	num, _ := strconv.Atoi(strings.TrimLeft(m[1], "0"))

	t := &Ticket{Num: num, Slug: m[2], Path: path}

	kv, body, ok := parseFrontmatter(src)
	if !ok {
		kv = map[string]string{}
		body = src
		t.Legacy = true
		for _, km := range reLegacyKey.FindAllStringSubmatch(src, -1) {
			key := strings.ToLower(km[1])
			if key == "blocked by" {
				key = "blocked_by"
			}
			kv[key] = strings.TrimSpace(km[2])
		}
	}

	if h := reH1.FindStringSubmatch(body); h != nil {
		t.Title = h[1]
	} else if h := reH1.FindStringSubmatch(src); h != nil {
		t.Title = h[1]
	}

	t.Type = Type(strings.ToLower(kv["type"]))
	t.Status = Status(strings.ToLower(strings.ReplaceAll(kv["status"], " ", "_")))
	t.ClaimedBy = kv["claimed_by"]
	t.ClaimedAt = kv["claimed_at"]
	t.Assets = splitList(kv["assets"])
	t.HasAnswer = reAnswer.MatchString(src)

	var err error
	if t.BlockedBy, err = splitNums(kv["blocked_by"]); err != nil {
		return nil, fmt.Errorf("blocked_by: %w", err)
	}
	if t.UnderminedBy, err = splitNums(kv["undermined_by"]); err != nil {
		return nil, fmt.Errorf("undermined_by: %w", err)
	}
	return t, nil
}

// section returns the body of a `## <name>` heading, up to the next `## `.
func section(src, name string) (string, int) {
	lines := strings.Split(src, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "## "+name {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", 0
	}
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			return strings.Join(lines[start:i], "\n"), start + 1
		}
	}
	return strings.Join(lines[start:], "\n"), start + 1
}

// bullets splits a section into top-level `- ` bullets with their line offsets,
// folding indented continuation lines into the bullet they belong to.
func bullets(body string, offset int) []struct {
	Text string
	Line int
} {
	var out []struct {
		Text string
		Line int
	}
	for i, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "- ") {
			out = append(out, struct {
				Text string
				Line int
			}{l, offset + i})
			continue
		}
		if len(out) > 0 && strings.TrimSpace(l) != "" && strings.HasPrefix(l, " ") {
			out[len(out)-1].Text += "\n" + l
		}
	}
	return out
}

func decisionsIn(src, name string) []Decision {
	body, off := section(src, name)
	var out []Decision
	for _, b := range bullets(body, off) {
		for _, m := range reDecision.FindAllStringSubmatch(b.Text, -1) {
			n, _ := strconv.Atoi(strings.TrimLeft(m[1], "0"))
			out = append(out, Decision{TicketNum: n, Line: b.Line})
		}
	}
	return out
}

func ParseMap(path, src string) *Map {
	m := &Map{Path: path}
	if h := reH1.FindStringSubmatch(src); h != nil {
		m.Name = h[1]
	}
	dest, _ := section(src, "Destination")
	m.Destination = strings.TrimSpace(dest)

	m.Decisions = decisionsIn(src, "Decisions so far")
	m.OutOfScope = decisionsIn(src, "Out of scope")

	fogBody, off := section(src, "Not yet specified")
	for _, b := range bullets(fogBody, off) {
		p := FogPatch{Line: b.Line}
		if tm := reFogTitle.FindStringSubmatch(b.Text); tm != nil {
			p.Title = strings.TrimRight(tm[1], ".")
		}
		if cm := reClearsRaw.FindStringSubmatch(b.Text); cm != nil {
			p.ClearsWith, _ = strconv.Atoi(strings.TrimLeft(cm[1], "0"))
		}
		out := p
		m.Fog = append(m.Fog, out)
	}
	return m
}
