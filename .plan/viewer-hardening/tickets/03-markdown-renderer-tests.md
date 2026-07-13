---
type: task
blocked_by: []
---

# Unit tests for the markdown renderer

## Question

`web/js/markdown.js` renders arbitrary ticket prose into the detail panel. It
became a pure importable module in the web/ refactor, so it can finally be
tested with `node --test` and zero dependencies — and it's the frontend code
most likely to regress subtly (escaping, nested emphasis around code spans,
fences quoting `## Answer`, cross-ticket link extraction).

Write that test file, covering at minimum: HTML escaping, bold/italic/inline
code and their nesting, fenced blocks (including markdown-looking content
inside them), ordered/unordered lists, blockquotes, hr, heading levels,
external links vs `data-goto` ticket links, and the empty-body fallback.
Decide where tests live and how they run (likely `node --test` invoked from
CI once the [CI syntax check](04-ci-module-syntax-check.md) exists).

## Answer

`cmd/wayfinder-maps/webtests/markdown.test.mjs` — 25 tests, `node --test
cmd/wayfinder-maps/webtests/*.test.mjs`, zero dependencies, now the final
step of CI's `frontend` job (globbed explicitly: Node 22 won't take a bare
directory as a `--test` target). Everything the ticket listed is covered, plus the renderer's
sharper corners: bold wrapping a code span, `*` inside code staying
unitalicised, seven hashes not being a heading, `***` reading as hr rather
than emphasis, an unclosed fence still flushing, list type switching mid-list,
and leading-zero ticket numbers collapsing in `data-goto`.

Two placement decisions worth keeping: tests live *outside* `web/` because
`//go:embed web` ships everything in that directory, and a two-line
`package.json` (`type: module`) at `cmd/wayfinder-maps/` makes Node parse the
imported `web/js/*.js` as ESM explicitly, instead of leaning on newer Node's
syntax auto-detection.
