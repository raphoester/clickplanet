# Design system

ClickPlanet looks like **a box of toy soldiers laid out on a night sky**: flat
colors, a thick ink outline around everything, hard drop shadows, and Luckiest
Guy for the game's own words. The globe is the one thing drawn in light and
shade; everything laid over it is a toy.

It was picked against nine other directions (pixel art, mission control, sports
broadcast, a passport, a war poster…) for one reason: it is the look the game
already had in its best part, the title medals, and nothing in it asks the
game to talk in another universe's words. Conquest, flags and army titles fit a
toy box; they do not fit a space station or a TV studio.

## Where it lives

- **`src/tokens.css`** holds every color, font, size, radius, outline and
  shadow, and nothing else. A value that is not there is not part of the game's
  look.
- **`src/index.css`** builds the shared pieces out of them: the panel, the box
  inside a panel, the buttons, the chips, the coins, the field.
- **A component's own CSS says only what makes that component itself**, and
  says it in tokens. It never restates a shared piece's size, radius or font
  (see "Styling" in `CLAUDE.md` for how that went wrong once).
- A shade the tokens lack is a mix, not a new literal:
  `color-mix(in srgb, var(--gold) 30%, transparent)`.
- **`src/designTokens.test.ts` holds all of this**: it fails on a color or a
  font family written anywhere but `tokens.css`.

## Type

Two families, and each has one job.

| Token | Face | For |
|---|---|---|
| `--font-display` | Luckiest Guy | The game's own words: the logo, panel headers, buttons, country names, counts on the board, title names, the big moments |
| `--font-text` | Rubik | Everything a player writes (usernames, messages), and every small or long text: labels, captions, rank lines, help, fields |

- **A player's words are never set in the display face.** Luckiest Guy has
  capitals only and Latin only: a Bulgarian username would come out in two
  fonts, half of it lowercase. Rubik covers Latin, Cyrillic, Hebrew and Arabic.
- **Sentence case, and no letter-spaced capitals.** Tracked uppercase labels in
  a condensed face are what made the old menus read as a generic dashboard.
- **Sizes are tokens**: `--text-xs` to `--text-l` for Rubik, `--display-s` to
  `--display-xxl` for Luckiest Guy. Nothing under 12px.
- **A text field is 16px or more**: Safari zooms the page in on a smaller one and
  never zooms back out.
- **Luckiest Guy rides high in its box.** Anything centred beside it is aligned
  on its capitals (`titleFont.ts`, `CountryFlag`), and a button in the display
  face gets about 3px more padding on top than below.

## Color

Each color has one job. Using one for another job is the eclecticism this
system exists to stop.

| Tokens | Job |
|---|---|
| `--ground`, `--ground-glow` | The sky behind the globe and the plain pages |
| `--panel` | Every panel |
| `--panel-raised` | A box inside a panel |
| `--panel-sunk` | A well: a progress track, a meter |
| `--ink` | Every outline and every drop shadow. Never black, never a color's own dark shade |
| `--text`, `--text-soft`, `--text-muted`, `--text-faint` | Text, from the main line down to what is locked or off |
| `--action` | The one thing to press on a screen (sign in, send, wear), and the player's own row |
| `--secondary` | Every other button |
| `--gold` | Reward and progress: panel headers, title names, progress fills, first place |
| `--gain`, `--loss` | Tiles won and lost, nowhere else |
| `--streak` | The streak flame |
| `--refill`, `--spread`, `--bomb`, `--enclose` (and `-light`) | A bonus kind: its box, its slot, its reward line |
| `--bronze` … `--holo-4`, `--conquest` … `--og-enamel` | Medals only (below) |

- **A name's color is only a hue.** `hueOf` reads it off the pick; the saturation
  and the lightness are tokens (`--author-*`), so no pick can be unreadable on a
  panel. A guest, or a player with no pick, has `--author-chroma: 0` and is grey.
- **Other companies' colors follow their own rules**: the Google and Discord
  sign-in buttons keep their brand colors, in their components.
- **The globe is not part of the system.** The earth, the flags, the tile
  effects and the bonus box in the sky keep their own colors in `app/viewer/`.
  The system starts at the first DOM element over the canvas.

## Shape

- **Every piece has an ink outline**: `--outline` (3px) on panels, buttons,
  fields and medals, `--outline-thin` (2px) on small things: chips, coins, a box
  inside a panel.
- **Shadows are hard, straight down, and ink**: `--drop-l` under a panel,
  `--drop-m` under a button, `--drop-s` under a small button, `--drop-xs` under a
  chip. No blur. A pressed button moves down by its drop and loses it.
- **A glow only says that something is happening** — a bonus switched on, a
  message arriving, a clock running out — and never decorates.
- **Radii**: `--radius-l` panels, `--radius-m` buttons, fields and boxes,
  `--radius-s` chips, `--radius-pill` pills and round buttons, `--radius-xs` the
  corner a chat balloon points from.
- **Display text on a panel stands on `--text-drop`**; the big lines (the logo,
  a title name, an unlock) wear `--text-outline`.
- **Spacing is in 4px steps**, `--space-1` to `--space-6`.

## Pieces

- **Panel**: `--panel`, the outline, `--radius-l`, `--drop-l`, a `--sheen` line
  along its top inside edge. Its header is display gold on `--text-drop`.
- **Buttons**: the action button (red, display face), the secondary button
  (blue), the gold button (wearing a title, taking a reward), and the icon
  button (a raised square). All outlined, all on a drop.
- **Chip**: a small pill on `--outline-thin`, for a count that changed, a tag,
  a stamp.
- **Coin**: the first three places on the board, in gold, silver and bronze with
  one hard shadow edge inside. The player's own place is a white coin with the
  number in `--action`.
- **Field**: white, ink text, the outline, `--radius-m`, 16px.
- **Balloon**: the author's pastel (`--author-balloon-lightness`), ink text, a
  thin ink outline, `--radius-m` with the `--radius-xs` corner.

## Medals

Every title is a medal, and the medals obey the same rules as everything else:

- **Flat metal**: bronze, silver, gold, platinum, and prism and holo as four flat
  bands. One hard light edge at the top left, one hard shadow edge at the bottom
  right, no gradient.
- **Ink**: an outline on the ring, the center and the icon, and a hard drop
  under the whole medal.
- **The track names the center and the ribbon**: Conquest navy and red, Chatter
  teal, Devotion brown and orange, OG violet.
- **The icon** is in the metal and `--cream`, outlined in ink.
- **A locked medal** is a grey ring, a sunk center and a padlock.

## Layout

**The globe is the game, and four zones are all that is drawn over it.** A new
feature goes in one of them; there is no fifth panel and no new fold.

| Zone | What goes there | Where |
|---|---|---|
| **Status** | Who you play for, its rank, the season's clock: chips that are read at a glance | Phone: the top bar. Desktop: the menu's head, the season at the top centre |
| **Moments** | Something that just happened: the quiz, a bomb, a caught box, native land. One at a time, then gone | Under the status zone |
| **Play** | What every click needs: the clicks left, the slowdown, the four bonuses | One bar, at the bottom |
| **Places** | Everything read or set: the board, the chat, the account, More. One open at a time | Phone: tabs at the bottom, each a sheet. Desktop: the menu's tabs on the left, the chat on the right |

- **Ask in this order.** A moment? Moments. Needed on every click? Play, and that
  is rare. Anything else is a place, or a line inside one.
- **A panel is as tall as what it holds**, up to the screen, and scrolls inside.
- **A line holds two things at most.** A third goes where it fits, a place.
- **The leader's frame holds the anthem**: the music is the leader's, so the
  player is on the board, not on the globe.

## Motion

Short and springy: a press lands in about 120ms, a panel opens in about 200ms.
Under `prefers-reduced-motion: reduce` movement goes and color stays — every
component already follows that rule and a new one does too.
