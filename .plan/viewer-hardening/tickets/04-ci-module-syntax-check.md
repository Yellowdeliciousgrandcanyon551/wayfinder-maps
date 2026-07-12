---
type: task
blocked_by: []
---

# CI guard for the embedded frontend

## Question

Since the web/ refactor, a syntax error in a JS module no longer fails
`go build` — it ships embedded in the binary and breaks only at runtime, in
the browser. The repo currently has no push/PR CI at all (only
`release.yml`), so nothing would catch it before a release.

Add a CI workflow that runs on push/PR: `go build ./...`, `go vet`, `go test
./...`, and a syntax pass over the frontend (`node --check` per module under
`web/js/`, or equivalent). Keep it minimal — the point is the embedded-JS
hole, not a full pipeline.

## Answer

`ci.yml` runs on push to main and on PRs, with two jobs. The `go` job does
`go build ./...`, `go vet ./...`, `go test ./...` with `CGO_ENABLED=0` — the
pure-Go build tags compile without the webkit2gtk headers release.yml has to
install, so the runner stays dependency-free. The `frontend` job syntax-checks
every module under `cmd/wayfinder-maps/web/js/`.

One trap found on the way: `node --check file.js` parses a `.js` file as
CommonJS and silently accepts broken `import`/`export` — a false guard for
exactly the failure this ticket exists to catch. The modules are instead fed
over stdin with `node --check --input-type=module`, which parses them as ESM
and actually fails on a broken module. Landed in 3997583 on the
`ci-embedded-frontend-guard` branch.
