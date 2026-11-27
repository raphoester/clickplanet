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

## Endless animations

**An endless animation moves only `opacity` or `transform`.** The compositor plays those on its own. A `box-shadow`,
a colour, a `filter` or a dash offset that is animated is a repaint of the element on every frame, on the main
thread, for as long as it is on screen. Some are on screen for days: a bonus held and never used wears its New glow
until it is used.

- **A glow is drawn once, on a pseudo-element, and only fades**: `.inventory-slot--new::before`,
  `.inventory-slot--on .inventory-box::before`, `.battle-chip.panel::after`, `.chat-waiting::before`. The pulse on the
  next title is a gold disc behind the medal that swells past its rim and fades. The box pointer runs through the
  kinds' colours as three layers, each fading in over the ones under it (`colourLayers` in `bonusPointer.ts`). The
  tap hand's hard shadow is its own silhouette under it, moved with the hand's transform and one more drop.
- **The battle chip clips nothing** (`overflow: hidden` would clip its glow): the tape's box is as tall as the
  panel's inner corner and rounds with it.
- **It plays as `var(--endless, running)`**, the last word of its `animation` shorthand. `:root[data-unwatched]`
  (in `index.css`) sets `--endless: paused`, and `globe.ts` puts `data-unwatched` on `<html>` a minute after the last
  input, the rule that stops the spin (see [Drawing only when something changed](render-on-demand.md)). The next input
  takes it off, and every animation goes on from where it stopped. A page with no globe never pauses.
- **A spinner the player waits on does not name it**: the sign-in callback's, the globe's loading status, the
  "results are coming" line (`battle-over-wait`) and the quiz's picked answer, which waits for the server. Nor does a
  finite animation: one that starts on a page nobody watches would stay on its first frame, often invisible.
- The marching dashes of an armed bonus with no end (`.inventory-loop`) still animate a dash offset: they show only
  while the player aims, and they pause with the rest.
- **`endlessAnimations.test.ts` holds all of this**: it fails on an endless animation that does not name
  `var(--endless, running)`, but the waits above, or whose keyframes move anything but `opacity` and `transform`, but
  the dashes and the picked answer.
- **Why.** Measured on 2026-11-19 in headless Chrome on an M2 Max, renderer and GPU process together, the globe still
  and drawing nothing, the page in use. A bomb held and never used: 21–23% of a core and 122 repaints a second before,
  9–12% and 2–4 after. The battle chip on a desktop screen: 26–31% and 120 repaints before, 11–15% and 2 after. The
  tap hand of a first visit on a desktop screen: 16–20% before, 10–12% after. On a page nobody watches, a held bomb:
  23–24% before, 7–8% after, and the compositor's frames fall from 60 a second to about 10.
- **The look is the same.** Each glow was shot at eight points of its cycle in both builds, every animation held at
  the same point: the mean difference is under 2 of 255 at every point. The slot's ring and the title's pulse grow
  in a little less at the start, as a fade does against a spread that grows.

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

**The scrolling body is `position: relative`**, so what is absolutely positioned
inside it is placed, clipped and scrolled by it. Without it, the board's rows'
`.sr-only` spans (the "points today" read aloud) took `.scroll-area` for their
containing block: the table did not clip them, and they stretched the panel's
own scroller, `.menu-body`, to the whole list's height. At the table's end the
wheel moved on to that scroller, which slid the leader's frame away over a blank
panel, then back (Nov 21 2026). Something meant to stay put over a body that
scrolls goes outside the `ScrollArea`, not on `.scroll-area`.

**A modal has `--space-5` on every side**, and its × sits in the corner at that
distance. Its scrollbar sits in that padding, so the text has as much room on
the right as on the left. The panel is 376px wide: a 328px column of text. A `Modal` given no `onClose` has no ×,
and Escape and the backdrop leave it open: the warning before an alt ban is one.

## Words in every language

**A box that holds a word is as wide as the word**, never a fixed width: the page speaks nineteen languages, and
German, Polish and Brazilian Portuguese run up to 30% longer than English ("Classement", "Compartilhar",
"Doładowanie"). Found by measuring every panel in every language at 360px, 390px and 1280px wide (Nov 2026, the first
nine). A side is `start` or `end`, never `left` or `right`, since Arabic mirrors the page
([Right to left](i18n.md#right-to-left)).

- **A row of tabs or slots** sizes each to its word and shares the room left: the desktop rail (`min-width`). The
  phone's tab bar has no words: its six tabs sized to "Classement" and "Chat" were uneven, so each is an icon alone,
  as wide as the others (`flex: 1 1 0`), named by its label and titled with it (Nov 2026). The dock has a fixed width, so a bonus slot keeps its 72px and only a
  longer name widens its own slot (the name carries the side padding); making every slot as wide as the widest
  pushed them over the clicks left.
- **A chip or a line that must stay one line** is never narrower than its words (`min-width: min-content`, the
  season chip).
- **A line that may get long** wraps only when it does not fit (`text-wrap: balance`): a hero line ("Nouveau
  titre !"), a news line with two long country names, the quiz banner. A label goes above its value when both do
  not fit beside a button (the phone's "Tu joues pour"), and the dock's "+1 dans 1s" goes under its "1,5× moins
  vite" chip when the wider bonus names leave no room beside it. The 1280px measure missed it: it ran with a full
  bank, where the countdown does not show.
- **A fixed column** (the board's numbers) keeps its width, and its head is a short word: the head's note in
  `notes.json` says so to the translator.

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
browser you have open. A phone on its side is `--w 750 --h 340` (an iPhone 14 in Safari) or `--w 568 --h 320` (the
smallest, an iPhone SE): the rail form of [The screen](screen.md). `--eval` evaluates in the page (top-level `await` works)
and prints the result — measuring boxes with `getBoundingClientRect()` beats
eyeballing a screenshot. `--headed`, `--w/--h/--dpr`, `--full` and `--settle`
cover the rest; `npm run mobile -- --help` lists them. It is a touch phone at every
`--w`: at 1440 wide, what shows only on hover (the chat's message buttons) shows as on
a phone. Judge a mouse desktop in the browser pane.

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

`--open-menu` opens the Board sheet. The asks never show there: they skip a
browser's first visit (see [Nudges](nudges.md)), and the throwaway profile is always
one.
