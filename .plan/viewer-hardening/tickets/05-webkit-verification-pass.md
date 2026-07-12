---
type: task
blocked_by: []
---

# WebKit verification pass

## Question

The viewer has only ever been verified in Chromium, and it now carries two
WebKit-specific code paths taken on trust: the label-alpha workaround in
`web/js/draw.js` (WebKit ignores canvas globalAlpha on shadowed fillText) and
the Safari `gesture*` pinch handling from
[touch and trackpad input](01-touch-and-trackpad-input.md), which Chromium
cannot even fire.

Run the app in WebKit and confirm: the map renders (stars, edges, fog,
labels), labels actually fade during map transitions, trackpad pinch zooms
the map rather than the page, and touch works on an iOS device or simulator.
Decide the tooling as part of the task — Playwright's WebKit build may cover
most of it headlessly (and could then join CI); whatever it can't reach
becomes a short manual Safari checklist documented in the verify skill.

## Answer

Playwright WebKit (26.5, `npx playwright-core install webkit`) covers nearly
everything headlessly, and the viewer passes: the full map renders identically
to Chromium (stars, glows, satisfied/unsatisfied edges, fog tethers, labels,
the undermined halo), the label-alpha workaround demonstrably works — a frame
captured 600ms into the fade-in shows labels dim *with* the constellation
rather than opaque over it — touch tap opens the panel, wheel zooms, and the
markdown body renders strong/code/lists/xlinks. Zero console errors.

Two things stay manual, now written into the verify skill's WebKit section:
real trackpad pinch (Safari's `gesture*` events — Playwright's WebKit cannot
construct a GestureEvent, so that path is code-reviewed plus manual), and
touch on an actual iOS device. The WebKit recipe itself also lives in the
verify skill, so future passes can join any CI or pre-release routine cheaply.
