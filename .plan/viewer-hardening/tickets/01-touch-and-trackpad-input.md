---
type: task
blocked_by: []
---

# Touch and trackpad input for the star-map

## Question

`web/js/input.js` wires mouse events only. On any touchscreen the map is
completely inert — no pan, no zoom, no way to open a ticket. On macOS
trackpads, pinch works in Chrome/Firefox by accident (it arrives as
ctrl+wheel) but in Safari pinch fires `gesture*` events instead, so it zooms
the page rather than the map.

Wire the missing input: one-finger drag pans, tap selects a star (with a
touch-sized hit radius), two-finger pinch zooms about the gesture midpoint
while tracking its pan, and Safari's gesture events drive the same zoom. The
browser's own touch panning/zooming must be suppressed on the canvas, and
mouse behavior — including wheel-to-zoom — must not change.

## Answer

Done in `web/js/input.js` plus one line of CSS (`touch-action:none` on the
canvas). One-finger drag pans through the same goal-easing path as the mouse;
a touch that moves under 8px is a tap and hit-tests with a 24px minimum radius
(fingers are wider than cursors). Two fingers become a pinch: zoom and
two-finger pan are folded into a single `zoomAt` step — rescale about the old
midpoint, land on the new one — and a pinch that collapses to one finger hands
off to a fresh drag rather than a stale one. Safari's nonstandard `gesture*`
events drive the same `zoomAt` (its `scale` is cumulative from gesturestart,
so the zoom anchors on the scale at start). The wheel handler was refactored
onto `zoomAt` too, unchanged in behavior.

Verified headless in Chromium via CDP touch events: tap opens the panel,
1-finger drag pans 1:1, pinch-out zooms in about the gesture centre, an
empty-space tap closes cleanly, and mouse wheel still zooms — zero console
errors. The Safari gesture path is code-reviewed only (Chromium cannot fire
`gesture*`); the WebKit verification pass in the map's fog covers it.
