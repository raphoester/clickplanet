# Styling

**Read [`DESIGN.md`](../DESIGN.md) before touching a style.** It is the design
system: what the game looks like, which token does which job, and the rules
every piece follows. The values themselves are in `src/tokens.css`, imported
first in `main.tsx`.

**`designTokens.test.ts` fails on a color or a font named anywhere else**: in a
stylesheet, in the home page's style, or in a component (the Google and Discord
sign-in colors aside). Code that draws outside CSS reads the tokens as well:
`drawShareCard.ts` with `getComputedStyle` at draw time, the medals through
`style`. The home page, `privacy.html` and `terms.html` link `src/tokens.css`
themselves.

Plain CSS files co-located with components. No CSS preprocessor or CSS-in-JS.

`ChatPanel.css` is the one file with a custom property contract: each message
and the folded peek carry `--author-hue` from `hueOf`, and the CSS builds
the author's stripe, name colour and arrival glow out of it. Keep the hue in the
TS and the rest in the CSS — that is what stops an author's colour from being
computed in two places with two different saturations.

`Leaderboard.css` has the other one: the delta badge's `animation-duration` is
set inline from `DELTA_HOLD_MS`, so the fade-out ends exactly as the badge is
dropped from the map. The keyframes are written so the badge **never dips below
where it comes to rest** — the rows are 25px and a badge that starts low lands
on the count below it.

`index.css` owns the shared boxes — `.button`, `.button-mini`, `.icon-button`,
`.menu-label` — **including their phone sizes** (the `COMPACT` query, see
[The screen](screen.md)). A component's own
CSS file says what makes that component itself (its colour, its icon spacing),
and must not restate height, padding, radius or font-size. Restating them is how
the old Discord button ended up a 56px slab next to a 40px About button on
mobile: its CSS set its own `height`, and the mobile rule it also carried
overrode the shared mobile size. Anchors styled as buttons (`BuyMeACoffee`, the
menu's Home link) legitimately need `display: flex` with both axes centred and
`text-decoration: none` — a `<button>` gets those for free — and nothing more.

## A panel that scrolls

**`components/ScrollArea.tsx` is the one scrolling body of a panel**: `Modal`,
`MenuPanel`, `Sheet` and the leaderboard's table all use it. It draws a 36px
fade over its bottom edge **only while more is below**: when everything fits,
and at the end of the scroll, there is no fade. A body that does not scroll
(`overflow: hidden`, as in the chat sheet and the share picture) has no fade
either. So no content pads its bottom to clear the fade, and no panel turns it
off.

Why: each panel drew its own fade at all times. The last line sat under it, so
each content added `padding-bottom: 36px` to escape it (the leaderboard in
#312, the account, players and About panels), and two modals turned it off. A
panel that missed the padding hid its last line: the sign-in modals' "Privacy
policy" (Oct 30 2026).

**A modal has `--space-5` on every side**, and its × sits in the corner at that
distance. Its scrollbar sits in that padding, so the text has as much room on
the right as on the left. The panel is 376px wide: a 328px column of text.

## Debugging mobile layout

**Do not check mobile by resizing the browser window.** macOS enforces a minimum
window width well above the 768px breakpoint, so the window stays wide (`window
.innerWidth` around 1900 on this machine), the mobile branch never engages, and
you end up inspecting a narrow desktop. Editing the media conditions in the
CSSOM to force the branch tests the rules but not the device: pointer, hover and
DPR all still say desktop.

Use `npm run mobile`, which drives Chrome's real device emulation over the
DevTools Protocol — the same `Emulation.*` calls the inspector's device toolbar
makes, so viewport, DPR, touch and the iOS user agent all resolve like a phone:

```bash
npm run dev
npm run mobile -- http://localhost:5173/play --open-menu --out /tmp/shot.png \
  --eval 'JSON.stringify(document.querySelector(".sheet").getBoundingClientRect())'
```

It runs headless Chrome under a throwaway profile, so it never disturbs the
browser you have open. `--eval` evaluates in the page (top-level `await` works)
and prints the result — measuring boxes with `getBoundingClientRect()` beats
eyeballing a screenshot. `--headed`, `--w/--h/--dpr`, `--full` and `--settle`
cover the rest; `npm run mobile -- --help` lists them.

Chrome's emulation does not reproduce one iOS behaviour that has bitten this
page: **Safari zooms the whole page in when a text field under 16px takes
focus**, and never zooms back out. That is why `.chat-input` is 16px — at 14px
the zoom pushed the send button off the right of the screen. Keep every `input`
here at 16px or more; the emulator will not tell you when one drops below.

Nor does it draw the browser's toolbars, so **`100vh` looks right in the
emulator and is not on a phone**: there it is the height with the toolbars
hidden, and this page never scrolls them away. A `Modal` sized `100vh` put the
bottom of the player card under the toolbar, with nothing left to scroll and a
drag that pulled the page to refresh. Size a full-screen layer with
`position: fixed; inset: 0` instead. `overscroll-behavior: none` on the root
stops pull-to-refresh everywhere.

`--open-menu` dismisses the donation modal and opens the Board sheet:
`DonationModal` rolls a coin on **every** load (`SHOW_PROBABILITY`), so without
it you will screenshot the donation modal half the time.
