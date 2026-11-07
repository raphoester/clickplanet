# A finger iOS took for its swipe back

**On an iPhone, a swipe from the left edge of the screen is Safari's "back".** When the swipe starts on the globe and
iOS takes it, then lets it go, the page gets the finger's `pointerdown` and never its `pointerup` or
`pointercancel` ([WebKit bug 239014](https://bugs.webkit.org/show_bug.cgi?id=239014), open since 2022, and not
something a page can turn off: Safari owns the gesture, and neither `touch-action` nor `overscroll-behavior`
reaches it).

**OrbitControls drops a finger only on those two events**, so it keeps following one that is gone. The next finger
makes two, and every gesture after reads as a pinch against the finger left at the edge: a drag up or down zooms, the
globe slides instead of turning, and a tap moves it under the finger before the click lands. It stayed that way until
a reload. Three.js has no fix of its own (checked up to 0.187).

**`endStrayFingers` (`src/app/viewer/strayFingers.ts`) ends such a finger for iOS.** The touch events still count the
fingers right, so it keeps the touch pointers pressed on the canvas (`Fingers`, in `src/domain/fingers.ts`, oldest
first) and holds them against that count:

- **A `touchstart` on the canvas**: a finger kept past `targetTouches.length` is stray, the oldest first. A finger's
  `pointerdown` comes before its `touchstart`, so the new finger is counted on both sides, and this ends the stray
  before the new one has moved: the first drag after the swipe already turns the globe. Were the order the other way
  round, the count would only lag, and end nothing that is still down.
- **A `touchend` or `touchcancel` anywhere with no touch left on the screen**: every finger kept is stray. Only this
  count of zero is trusted on a lift, because the lift's `pointerup` and `touchend` can come in either order.
- **`pagehide`, or the page hidden**: no finger stays down on a page put away.

**A stray finger is ended with the `pointercancel` iOS owed**, dispatched on the canvas, so OrbitControls and the
globe's own press trackers (`HoldToDrop`, `TilePeek`) let go of it as they would of the browser's.

**OrbitControls then releases the capture of the last finger it let go of, and releasing a pointer the browser has
already ended throws** `NotFoundError`. `endStrayFingers` puts a `releasePointerCapture` on the canvas that releases
only what the canvas holds, which is all a release could do anyway.

`strayFingers.test.ts` runs the real OrbitControls in jsdom through the swipe and the drag after it: without the fix,
that one-finger drag zooms.
