# wayfinder

A read-only CLI for [wayfinder](https://agentskills.io) maps — the markdown planning
memory an agent leaves under `.plan/<effort>/` as it charts a large effort as a graph
of investigation tickets.

It answers two questions without an LLM:

- **`wayfinder status <dir>`** — what is resolved, what is in flight, and what is ready
  to claim (the *frontier*: open, unclaimed, every blocker resolved).
- **`wayfinder lint <dir>`** — does the map still tell the truth?

```
$ wayfinder status ../expensif/.plan/daily-timeline
Daily Timeline — continuous days, empty days included
5 resolved · 0 claimed · 4 open · 0 out of scope

Frontier — ready to claim, first by number wins:
  04  Re-shape DailyGroups around dates, not expens…  grilling

Blocked:
  05  Should HandleDaily's two branches converge      waits on 04
  06  Contain the day-card chrome drift               waits on 01, 04
  07  The infinite-scroll island's contract           waits on 01, 04

Fog: 7 patches, 0 anchored to a ticket
```

## Why

The map is shared memory: concurrent and future agent sessions orient to it before
choosing what to work on. When it drifts, it lies to them silently. It has already
happened — a project's `AGENTS.md` read "4 of 9 tickets resolved" while the map said 5.

Frontmatter makes drift *detectable*. This tool makes it *not happen*. Nothing here is
clever; the value is that it runs.

## The format contract

Structure the facts, leave the prose alone. A ticket's status, type, and edges have one
correct value and a machine can check them. Its **Question**, its **Answer**, the map's
**Destination** and **Notes**, and the one-line gist in Decisions-so-far are prose, are
lossy by design, and are never parsed. Structure those and you have built a Jira.

A ticket is `.plan/<effort>/tickets/NN-<slug>.md`:

```markdown
---
type: research | prototype | grilling | task
status: open | claimed | resolved | out_of_scope
blocked_by: [02, 03]                  # [] when none
claimed_by: <session id>              # when status: claimed
claimed_at: 2026-07-10T09:00:00Z      # RFC 3339, when status: claimed
undermined_by: [08]                   # optional — resolved on a premise 08 later broke
assets: [../assets/x.html.approved]   # optional
---

# <Ticket title>

## Question
…
## Answer          # present iff status is resolved or out_of_scope
…
```

Three fields carry weight beyond bookkeeping. **`out_of_scope`** is a status, not a flavour
of resolved — the skill keeps scope boundaries out of Decisions-so-far because they are not
steps on the route, so a parser that lumps them in over-counts. **`undermined_by`** marks a
decision whose premise a later ticket broke; without it a renderer paints the node green and
launders a live problem into a checkmark. **`claimed_at`** is what separates a dead session
from live work.

Fog patches in the map's `## Not yet specified` are bullets with a bolded lead title and an
optional anchor — enough to give a patch identity and, when known, a position:

```markdown
- **Today's row.** Whether today is visually distinguished. <clears-with: 04>
```

Fog stays deliberately coarse. It is not promoted to one file per patch, because the skill
says to write it as loosely as the view allows.

## Checks

Errors: dangling or self `blocked_by`; `blocked_by` cycles; duplicate ticket numbers (two
parallel sessions both reaching for `10`); closed without an `## Answer`; a resolved ticket
missing from Decisions-so-far; an out-of-scope ticket listed as a decision, or missing from
Out-of-scope; Decisions-so-far pointing at a ticket that is not resolved; a fog patch
duplicating a live ticket's title, or anchored to a ticket already resolved; a missing
Destination; unknown type or status.

Warnings: loose (pre-frontmatter) headers; a claim with no owner or older than 72h; an
`## Answer` on an open ticket; a ticket blocked by something ruled out of scope, which can
therefore never unblock.

Exit `0` clean, `1` errors found, `2` no map at that path.

## Reading legacy maps

The parser accepts the older loose header (`Type:` / `Status:` / `Blocked by:` after the H1)
and lints it as a warning, so it runs against maps written before the frontmatter existed.

## Build

```
go build ./cmd/wayfinder      # no dependencies
go test ./...
```

The module path is bare (`module wayfinder`) for local use. Give it a real path before
`go install`ing it from a remote.

## Not built yet

The GUI. The parser exists so a canvas *could* render the graph, with `undermined_by` nodes
amber and fog anchored where `clears-with` says. That is worth doing once the linter has run
clean across a couple of real efforts — and not before, because a canvas built on a format
that still drifts just renders the drift.
