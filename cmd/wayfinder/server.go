package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"

	"wayfinder/internal/wayfinder"
)

// serve runs the map viewer as a local web server, blocking until interrupted.
func serve(dir string) int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "7777"
	}
	addr := "localhost:" + port
	fmt.Printf("wayfinder: serving %s at http://%s\n", dir, addr)
	if err := http.ListenAndServe(addr, newServer(dir)); err != nil {
		fmt.Fprintf(os.Stderr, "wayfinder: %v\n", err)
		return 2
	}
	return 0
}

// newServer builds a handler that re-reads the effort on every request, so a
// saved edit shows on the next refresh. dir is the effort directory (map.md +
// tickets/); the map is never cached, because the whole point of the view is to
// mirror the files as they change.
func newServer(dir string) http.Handler {
	mux := http.NewServeMux()

	load := func(w http.ResponseWriter) (*wayfinder.Effort, bool) {
		e, err := wayfinder.Load(dir)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errorTmpl.Execute(w, err.Error())
			return nil, false
		}
		return e, true
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		e, ok := load(w)
		if !ok {
			return
		}
		if err := mapTmpl.Execute(w, buildMapView(e)); err != nil {
			fmt.Fprintf(os.Stderr, "wayfinder: render: %v\n", err)
		}
	})

	mux.HandleFunc("/ticket", func(w http.ResponseWriter, r *http.Request) {
		e, ok := load(w)
		if !ok {
			return
		}
		num, _ := strconv.Atoi(r.URL.Query().Get("n"))
		t := e.ByNum(num)
		if t == nil {
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile(t.Path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errorTmpl.Execute(w, err.Error())
			return
		}
		if err := ticketTmpl.Execute(w, buildTicketView(e, t, string(body))); err != nil {
			fmt.Fprintf(os.Stderr, "wayfinder: render: %v\n", err)
		}
	})

	return mux
}

// --- view models ---------------------------------------------------------

// Pixel geometry for the graph. Nodes are laid out on a grid of ranks (columns)
// and within-rank slots (rows); edges are beziers between the port on a
// blocker's right edge and the port on its dependent's left edge.
const (
	colW  = 260
	rowH  = 104
	nodeW = 210
	nodeH = 72
	padX  = 40
	padY  = 40
)

type nodeView struct {
	Num        int
	Title      string
	Type       string
	Status     string // css class: resolved|claimed|open|blocked|out_of_scope
	Glyph      string
	Frontier   bool
	Undermined bool
	Claimed    string
	X, Y       int
}

type edgeView struct {
	Path string // an SVG path "d" attribute
}

type mapView struct {
	Name          string
	Resolved      int
	Claimed       int
	Open          int
	OutOfScope    int
	Total         int
	Width, Height int
	Nodes         []nodeView
	Edges         []edgeView
	Decisions     []decisionView
	Fog           []fogView
	Empty         bool // no tickets at all
}

type decisionView struct {
	Num   int
	Title string
}

type fogView struct {
	Title      string
	ClearsWith int
}

func glyph(s wayfinder.Status) string {
	switch s {
	case wayfinder.StatusResolved:
		return "✓"
	case wayfinder.StatusClaimed:
		return "◆"
	case wayfinder.StatusOutOfScope:
		return "⊘"
	default:
		return "○"
	}
}

func buildMapView(e *wayfinder.Effort) mapView {
	name := e.Map.Name
	if name == "" {
		name = e.Name
	}
	v := mapView{
		Name:       name,
		Resolved:   e.Count(wayfinder.StatusResolved),
		Claimed:    e.Count(wayfinder.StatusClaimed),
		Open:       e.Count(wayfinder.StatusOpen),
		OutOfScope: e.Count(wayfinder.StatusOutOfScope),
		Total:      len(e.Tickets),
		Empty:      len(e.Tickets) == 0,
	}

	frontier := map[int]bool{}
	for _, t := range e.Frontier() {
		frontier[t.Num] = true
	}

	layers := e.Layers()
	pos := make(map[int][2]int, len(e.Tickets)) // ticket num -> top-left x,y

	for li, layer := range layers {
		x := padX + li*colW
		for ri, t := range layer {
			y := padY + ri*rowH
			pos[t.Num] = [2]int{x, y}

			status := string(t.Status)
			if t.Status == wayfinder.StatusOpen && !frontier[t.Num] {
				status = "blocked"
			}
			v.Nodes = append(v.Nodes, nodeView{
				Num:        t.Num,
				Title:      t.Title,
				Type:       string(t.Type),
				Status:     status,
				Glyph:      glyph(t.Status),
				Frontier:   frontier[t.Num],
				Undermined: len(t.UnderminedBy) > 0,
				Claimed:    t.ClaimedBy,
				X:          x,
				Y:          y,
			})
		}
		if x+nodeW+padX > v.Width {
			v.Width = x + nodeW + padX
		}
		if h := padY + len(layer)*rowH; h > v.Height {
			v.Height = h
		}
	}

	// Edges: from each blocker's right-middle to the dependent's left-middle.
	for _, t := range e.Tickets {
		to, ok := pos[t.Num]
		if !ok {
			continue
		}
		for _, b := range t.BlockedBy {
			from, ok := pos[b]
			if !ok {
				continue
			}
			x1, y1 := from[0]+nodeW, from[1]+nodeH/2
			x2, y2 := to[0], to[1]+nodeH/2
			c := (x2 - x1) / 2
			v.Edges = append(v.Edges, edgeView{
				Path: fmt.Sprintf("M%d,%d C%d,%d %d,%d %d,%d", x1, y1, x1+c, y1, x2-c, y2, x2, y2),
			})
		}
	}

	for _, d := range e.Map.Decisions {
		dv := decisionView{Num: d.TicketNum}
		if t := e.ByNum(d.TicketNum); t != nil {
			dv.Title = t.Title
		}
		v.Decisions = append(v.Decisions, dv)
	}
	for _, f := range e.Map.Fog {
		v.Fog = append(v.Fog, fogView{Title: f.Title, ClearsWith: f.ClearsWith})
	}
	return v
}

type ticketView struct {
	Num       int
	Title     string
	Type      string
	Status    string
	BlockedBy []blockRef
	Body      string
}

type blockRef struct {
	Num      int
	Resolved bool
}

func buildTicketView(e *wayfinder.Effort, t *wayfinder.Ticket, body string) ticketView {
	v := ticketView{
		Num:    t.Num,
		Title:  t.Title,
		Type:   string(t.Type),
		Status: string(t.Status),
		Body:   body,
	}
	for _, b := range t.BlockedBy {
		ref := blockRef{Num: b}
		if dep := e.ByNum(b); dep != nil {
			ref.Resolved = dep.Status == wayfinder.StatusResolved
		}
		v.BlockedBy = append(v.BlockedBy, ref)
	}
	return v
}

// --- templates -----------------------------------------------------------

var funcs = template.FuncMap{
	"pct": func(n, total int) int {
		if total == 0 {
			return 0
		}
		return n * 100 / total
	},
}

var mapTmpl = template.Must(template.New("map").Funcs(funcs).Parse(pageCSS + `
<div class="wrap">
  <header>
    <h1>{{.Name}}</h1>
    <div class="counts">
      <span class="c resolved">{{.Resolved}} resolved</span>
      <span class="c claimed">{{.Claimed}} claimed</span>
      <span class="c open">{{.Open}} open</span>
      {{if .OutOfScope}}<span class="c oos">{{.OutOfScope}} out of scope</span>{{end}}
    </div>
    {{if .Total}}<div class="bar"><span style="width:{{pct .Resolved .Total}}%"></span></div>{{end}}
  </header>

  {{if .Empty}}
    <p class="muted">No tickets yet.</p>
  {{else}}
  <div class="graph" style="width:{{.Width}}px;height:{{.Height}}px">
    <svg width="{{.Width}}" height="{{.Height}}" class="edges">
      {{range .Edges}}<path d="{{.Path}}"/>{{end}}
    </svg>
    {{range .Nodes}}
    <a class="node {{.Status}}{{if .Frontier}} frontier{{end}}" href="/ticket?n={{.Num}}"
       style="left:{{.X}}px;top:{{.Y}}px">
      <div class="node-top">
        <span class="num">{{printf "%02d" .Num}}</span>
        <span class="glyph">{{.Glyph}}</span>
        {{if .Frontier}}<span class="tag frontier-tag">frontier</span>{{end}}
        {{if .Undermined}}<span class="tag warn">undermined</span>{{end}}
      </div>
      <div class="node-title">{{.Title}}</div>
      <div class="node-type">{{.Type}}{{if .Claimed}} · {{.Claimed}}{{end}}</div>
    </a>
    {{end}}
  </div>

  <div class="legend">
    <span class="chip resolved">✓ resolved</span>
    <span class="chip claimed">◆ claimed</span>
    <span class="chip frontier">○ frontier</span>
    <span class="chip blocked">○ blocked</span>
    <span class="chip oos">⊘ out of scope</span>
    <span class="chip warn">undermined</span>
  </div>
  {{end}}

  <div class="cols">
    <section>
      <h2>Decisions so far</h2>
      {{if .Decisions}}<ul class="decisions">
        {{range .Decisions}}<li><a href="/ticket?n={{.Num}}"><span class="num">{{printf "%02d" .Num}}</span> {{.Title}}</a></li>{{end}}
      </ul>{{else}}<p class="muted">None yet.</p>{{end}}
    </section>
    <section>
      <h2>Fog — not yet specified</h2>
      {{if .Fog}}<ul class="fog">
        {{range .Fog}}<li>{{.Title}}{{if .ClearsWith}} <span class="anchor">clears with {{printf "%02d" .ClearsWith}}</span>{{end}}</li>{{end}}
      </ul>{{else}}<p class="muted">Clear.</p>{{end}}
    </section>
  </div>
</div>
`))

var ticketTmpl = template.Must(template.New("ticket").Funcs(funcs).Parse(pageCSS + `
<div class="wrap">
  <a class="back" href="/">← map</a>
  <header class="ticket-head">
    <h1><span class="num">{{printf "%02d" .Num}}</span> {{.Title}}</h1>
    <div class="counts">
      <span class="c {{.Status}}">{{.Status}}</span>
      <span class="c open">{{.Type}}</span>
      {{range .BlockedBy}}<span class="c {{if .Resolved}}resolved{{else}}blocked{{end}}">blocked by {{printf "%02d" .Num}}</span>{{end}}
    </div>
  </header>
  <pre class="body">{{.Body}}</pre>
</div>
`))

var errorTmpl = template.Must(template.New("err").Parse(pageCSS + `
<div class="wrap">
  <h1>Couldn't read the map</h1>
  <pre class="body err">{{.}}</pre>
  <p class="muted">Fix the effort directory and refresh.</p>
</div>
`))

const pageCSS = `<style>
  :root{
    --bg:#0f1115; --panel:#171a21; --line:#262b36; --ink:#e6e9ef; --muted:#8b93a3;
    --resolved:#3fb27f; --claimed:#e0a44b; --frontier:#4f8cff; --blocked:#565d6b; --oos:#5a5560; --warn:#e06c75;
  }
  *{box-sizing:border-box}
  body{margin:0;background:var(--bg);color:var(--ink);
    font:14px/1.5 ui-sans-serif,-apple-system,Segoe UI,Roboto,sans-serif}
  .wrap{max-width:1100px;margin:0 auto;padding:28px 24px 64px}
  h1{font-size:20px;margin:0 0 10px}
  h2{font-size:13px;text-transform:uppercase;letter-spacing:.06em;color:var(--muted);margin:0 0 10px}
  .counts{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:12px}
  .c{font-size:12px;padding:2px 9px;border-radius:999px;border:1px solid var(--line);color:var(--muted)}
  .c.resolved{color:var(--resolved);border-color:var(--resolved)}
  .c.claimed{color:var(--claimed);border-color:var(--claimed)}
  .c.open,.c.frontier{color:var(--frontier);border-color:var(--frontier)}
  .c.blocked{color:var(--blocked)}
  .c.out_of_scope,.c.oos{color:var(--oos)}
  .bar{height:6px;background:var(--line);border-radius:999px;overflow:hidden;max-width:420px}
  .bar>span{display:block;height:100%;background:var(--resolved)}
  .graph{position:relative;margin:22px 0;overflow:auto}
  .edges{position:absolute;left:0;top:0;pointer-events:none}
  .edges path{fill:none;stroke:var(--line);stroke-width:2}
  .node{position:absolute;width:210px;height:72px;padding:9px 11px;border-radius:10px;
    background:var(--panel);border:1px solid var(--line);text-decoration:none;color:var(--ink);
    display:flex;flex-direction:column;gap:3px;transition:transform .08s,border-color .08s}
  .node:hover{transform:translateY(-2px);border-color:var(--frontier)}
  .node-top{display:flex;align-items:center;gap:7px}
  .num{font-variant-numeric:tabular-nums;font-weight:700;color:var(--muted)}
  .glyph{margin-left:auto}
  .node.resolved{border-left:3px solid var(--resolved)}
  .node.resolved .glyph{color:var(--resolved)}
  .node.claimed{border-left:3px solid var(--claimed)}
  .node.claimed .glyph{color:var(--claimed)}
  .node.blocked{opacity:.72;border-left:3px solid var(--blocked)}
  .node.out_of_scope{opacity:.5;border-left:3px solid var(--oos)}
  .node.out_of_scope .node-title{text-decoration:line-through}
  .node.frontier{border-left:3px solid var(--frontier);box-shadow:0 0 0 2px rgba(79,140,255,.35)}
  .node.frontier .glyph{color:var(--frontier)}
  .node-title{font-size:12.5px;line-height:1.25;overflow:hidden;
    display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical}
  .node-type{font-size:11px;color:var(--muted);margin-top:auto}
  .tag{font-size:10px;padding:1px 6px;border-radius:999px}
  .tag.frontier-tag{background:rgba(79,140,255,.16);color:var(--frontier)}
  .tag.warn{background:rgba(224,108,117,.16);color:var(--warn)}
  .legend{display:flex;gap:14px;flex-wrap:wrap;margin:6px 0 30px;font-size:12px;color:var(--muted)}
  .chip.resolved{color:var(--resolved)} .chip.claimed{color:var(--claimed)}
  .chip.frontier{color:var(--frontier)} .chip.blocked{color:var(--blocked)}
  .chip.oos{color:var(--oos)} .chip.warn{color:var(--warn)}
  .cols{display:grid;grid-template-columns:1fr 1fr;gap:28px;margin-top:8px}
  @media(max-width:720px){.cols{grid-template-columns:1fr}}
  section{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:16px 18px}
  ul{list-style:none;margin:0;padding:0;display:flex;flex-direction:column;gap:8px}
  .decisions a{color:var(--ink);text-decoration:none}
  .decisions a:hover{color:var(--frontier)}
  .fog li{color:var(--muted)}
  .anchor{font-size:11px;color:var(--frontier);border:1px solid var(--line);padding:1px 6px;border-radius:999px;margin-left:4px}
  .muted{color:var(--muted)}
  .back{color:var(--muted);text-decoration:none;font-size:13px}
  .back:hover{color:var(--frontier)}
  .ticket-head{margin-top:14px}
  .body{background:var(--panel);border:1px solid var(--line);border-radius:12px;padding:18px 20px;
    white-space:pre-wrap;word-wrap:break-word;font:13px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace;
    color:var(--ink);overflow-x:auto}
  .body.err{color:var(--warn)}
</style>
`
