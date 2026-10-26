# Data flow

1. `useGlobe` calls `createGlobe`, which fetches the landmask (and rebuilds the
   tile positions from it, see [Tile positions](tile-positions.md)) and the borders
   blobs — in parallel, and before allocating any GPU resource, so an abandoned
   load never opens a context.
2. Ownerships are fetched in batches, eight at a time (see
   [`src/backends/`](backends.md)), and each is fed to `TileOwnership` as it
   lands. **`createGlobe`
   resolves only once the last batch is in**, so `useGlobe` stays `loading` —
   with `territories`, the share fetched, for the progress bar — and the menu,
   the leaderboard and every other control stay hidden until then. The scene
   already turns and fills in behind the loader, but a click claims nothing
   before the map is complete: an empty map with an empty board is not a game.
   A fetch that fails after its retries fails the whole globe, and an abandoned
   one disposes it. On ready, `publishLeaderboard` shows the full board at once
   rather than up to a sample later. Each batch brings its tiles' shields
   too (see [Shields](shields.md)).
3. Live updates arrive over the `ListenForEvents` stream, batched every 100 ms, into the same
   store.
4. Whatever the store reports as changed is painted, and the leaderboard is
   re-ranked from its counts — then handed to `useLeaderboardFeed`, which
   publishes it to React twice a second rather than ten times.
5. A click paints optimistically and POSTs; the server's echo confirms it later.
   A refused click is taken back off the map. A throttled one bumps `refusals` in
   `useGlobe`, and `ClickBudgetMeter` shakes and flashes red once per bump — a
   dialog here was annoying, since a player hits the wall mid-burst and the meter
   already says why. The other three raise a flag that `Viewer` renders as
   `BannedModal`, `VPNBlockedModal` or `SessionUnavailableModal`. The globe reports
   every refused click, so those flags hold one value and are not a queue — a burst
   is one thing to say, once.
   `reportClickFailure` in `globe.ts` is the five-way branch that picks which,
   split out of the click handler because it is the one piece of that handler
   worth testing: sending a refusal to the wrong dialog leaves a working page
   giving the wrong advice.

   Dismissing `VPNBlockedModal` only closes it. Unlike a spent bucket, that
   refusal does not clear on its own — the player has to change network — so the
   next click raises it again.

   **`BannedModal` says until when** (`describeEnd`, the chat mute's words: the
   time under 20 hours, then the day, and the year when it is not this one), and
   links the terms' "Play fair" section in a new tab (`GAME_RULES`,
   `/terms#play-fair`). It gives no reason, on purpose: the server sends none.
   Bombs and shields refused for a ban raise it too (`reportClaimFailure`).

6. `ClickBudgetMeter` shows what is left of the bucket, in the dock at the bottom
   (see [The screen](screen.md)). It warns
   before the wall, and shakes when a click hits it. **Its shape is read off the server's policy**: one
   pip per click in the burst (one bar past 12 of them), and the partly-filled
   pip is the click being granted back, at the server's own rate. Change
   the rule set's `clicks.burst` on the backoffice and this follows with no release here.

   **A slow refill gets a countdown.** When a click takes 1.5s or more to come
   back (`COUNTDOWN_FROM_S`; production is one every 5s), the meter says
   "+1 in 4s" under the bar, and says nothing at a full bucket. Past 12 pips
   the bar moves a sixtieth per click, too little to see, so a blue strip under
   it (`--click-budget-next`) fills once per click, as a pip would.

   The refill is animated from **one CSS custom property written per frame**,
   and each pip works out its own share of it with a `clamp()`; the count, the
   colour and the aria value are written only when the whole number changes.
   `useClickBudget` therefore re-renders when the *server* says something, not
   as the bucket refills — this sits beside a WebGL scene that wants the main
   thread. Under `prefers-reduced-motion` the fill steps four times a second
   instead of gliding.

   **It says when the toll slows the refill**: a chip before the countdown,
   "4× slower" (on a phone "4×", named in full), whenever `ClickBudget.price`
   says a slowdown above 1. The chip, the countdown and the offer below are a
   line beside the meter, never inside it.

   **The reading opens "Your clicks"** (`ClicksPanel`): a button laid over the
   reading (`.click-budget-open`), so `role="meter"` stays a leaf. On a desktop it
   is a popover over the dock, closed by the reading or Escape; on a phone, a
   sheet. It holds the count, the country's share of the map, the whole toll
   table from `BonusRules.toll` with the step the country is on marked, who else
   spends from the bank, and the offer. **It shows no seconds per click**: the
   reading's rate is the pace of the last click, which a change of flag only
   moves at the next click (see the backend's toll), so a time worked out from it
   would be wrong exactly when the player is looking.

   **A guest is offered to click faster.** A signed-in account refills
   `ClickBudget.linkedMultiplier` times faster (2 in production), into a bank of
   the same size. For a guest the
   server offers sign-in to, `Viewer` passes `onSignIn` and the dock shows
   "Sign in: clicks 2× faster" in its line — a button beside the meter, not in
   it, since the meter is a reading. It glows when the bucket is empty or a click
   is refused, the moment a guest meets the wall. **It is in the line only where
   it fits**: on a desktop with no slowdown chip. Otherwise it is in "Your
   clicks" and the Sign in tab, so the line never holds more than two things. It opens `SignInPitchModal`,
   which has the account panel's sign-in buttons. **The pitch sells more than
   speed**: a list of what a guest does not have — the clicks, a name, a color
   (guests are grey), a place on the board (players with a name are listed above
   the guests) and a streak flame (guests have none). Keep each line true: it names what
   the game does today, not what is planned. The account panel's guest text
   says the same. **Nothing is offered without the server's number**, nor with
   sign-in off.

   **It says when somebody else spends from the bucket.** The guests behind one
   address share one bank, and every player behind it shares the scope's, so a
   count can drop by clicks this player never made: another tab, or a stranger
   on the same carrier. `ClickBudget.sharedWith` is the server's answer to whose
   bucket the reading is: the dock's line shows a players icon named "Shared with
   the guests on your network" or "…everyone on your network", and "Your clicks"
   says it in words. Absent, it is the
   player's own and nothing is said. A guest who shares is offered "Sign in:
   your own clicks" instead, and `SignInPitchModal` says why.

   **It is one panel, and its width is set rather than grown.** The reading, its
   line and the inventory all live in `.click-budget-dock`, one row at the bottom
   centre that carries the only border and background; `.click-budget` itself
   draws nothing, so `role="meter"` stays a leaf with no button inside it. The
   dock's width is `--dock-width` (620px, 460px under 1280px, the screen less 24px
   on a phone): shrink-to-fit handed the widest part the say before, and the
   others trailed dead space. The reading is `flex: 1` and the slots keep their
   size, so the bar takes what is left. The panel's border is what carries state: the inventory's glow
   first, then low, empty and refused, in that source order so red at the wall
   beats a bonus being switched on. The reading still jolts on a refusal
   (`.click-budget-refused`, taken off on its own `animationend`), but the red
   flash is the dock's, through `:has()`.
