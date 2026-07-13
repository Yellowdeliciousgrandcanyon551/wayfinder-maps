# Star-map viewer hardening

## Destination

The maps viewer is first-class on every input device (mouse, trackpad, touch)
and the newly modular frontend under `cmd/wayfinder-maps/web/` cannot silently
regress — its sharp edges warned about, its pure logic tested, its syntax
checked before anything ships embedded.

## Notes

This effort carries execution: its tickets do the work, not just decide it.
The viewer's code lives in `cmd/wayfinder-maps/` (Go server + vanilla-JS ES
modules under `web/`, no build step — a constraint to preserve). Verify changes
end-to-end with the repo's `verify` skill (`.claude/skills/verify/SKILL.md`),
which documents how to build, serve a fixture map, and drive the app headless.

## Decisions so far

<!-- one line per resolved ticket: gist + link -->

- [Touch and trackpad input for the star-map](./tickets/01-touch-and-trackpad-input.md) —
  one-finger pan, fat-finger tap, pinch (and Safari gesture events) all drive
  the existing camera goal through a shared `zoomAt`; wheel behavior unchanged.
- [Warn when WAYFINDER_DEV points at nothing](./tickets/02-wayfinder-dev-missing-dir-warning.md) —
  startup stderr warning naming the resolved path when the dev dir lacks
  index.html; advisory, so serving proceeds and the embedded path stays silent.
- [WebKit verification pass](./tickets/05-webkit-verification-pass.md) —
  Playwright WebKit passes everything it can reach (render, label fade, touch
  tap, wheel, markdown); trackpad pinch and real-iOS touch stay on a manual
  Safari checklist in the verify skill.
- [Unit tests for the markdown renderer](./tickets/03-markdown-renderer-tests.md) —
  25 zero-dependency `node --test` cases in `cmd/wayfinder-maps/webtests/`
  (outside `web/`, so nothing lands in the embedded binary), run as the final
  step of CI's frontend job; a `type: module` package.json scopes the modules
  as ESM explicitly.
- [CI guard for the embedded frontend](./tickets/04-ci-module-syntax-check.md) —
  `ci.yml` on push/PR runs `go build`/`vet`/`test` (CGO_ENABLED=0, so no
  webkit headers) plus an ESM syntax pass over `web/js/`; the modules go through
  `node --check --input-type=module` over stdin, since `node --check file.js`
  parses as CommonJS and silently accepts broken `import`/`export`.

## Not yet specified

- **Whether the frontend's ES5-style JS modernizes.** The modules kept the old
  `var`-and-concatenation style verbatim through the refactor; module-by-module
  modernization is possible now, but it's not yet clear it's worth the churn.

## Out of scope

- **Editing from the viewer** — claiming or resolving tickets in the panel is
  ruled out for now: all writes go through files, which keeps agents and humans
  on equal footing and the viewer honest as a pure lens on the map.
