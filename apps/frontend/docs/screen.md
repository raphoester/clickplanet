# The screen: four zones

**Nothing is drawn over the globe but four zones**, and a new feature goes in one
of them rather than in a fifth panel. The rule is in [`DESIGN.md`](../DESIGN.md);
this is where each zone lives. `Viewer` composes them, and `useCompact`
(`compact.ts`, the 768px query, kept live) picks the phone or the desktop form.

| Zone | Phone | Desktop |
|---|---|---|
| **Status** | `hud/StatusBar`: logo, flag, country and rank (opens the board), `SeasonChip` at its end | `Menu`'s header and "playing for", `SeasonChip` at the top centre |
| **Moments** | under the status bar (`--status-bottom`) | under the season chip |
| **Play** | the dock (`ClickBudgetMeter` + `Inventory`) above the tab bar, the chat's peek above it | the dock at the bottom centre |
| **Places** | `hud/TabBar` (Board, Chat, Sign in / You, Share, Settings), each a `hud/Sheet` | `Menu`'s tabs (Board, You, Share, Settings) on the left, the chat on the right |

- **One sheet at a time on a phone.** `Viewer` holds which (`sheet`): the five
  tabs, and the season and "Your clicks", which the status bar and the dock open.
  A tab pressed again closes it. The sheets are the same places the desktop
  shows: `BoardPlace` and `YouPlace` in `Menu.tsx`, `SharePlace`, `SettingsPlace`,
  `SeasonDetails`, `ClicksPanel`, and the chat's own sheet.
- **A tab opens a place, and no tab is an action.** The camera is a tile in
  Share, not a tab of its own.
- **There is no More.** It held eight tiles of four kinds, and each new feature
  with no home went there. Each kind has a home now: the picture and the recruit
  link in Share (see [Sharing the globe](sharing.md)), the feedback form
  and About at the foot of Settings, Discord, TikTok and Instagram in the chat's
  bar and in About. A new entry goes to the place of its kind.
- **Settings is every switch the player keeps**: the display (`useDisplaySettings`,
  `domain/displaySettings.ts`, in `clickplanet-display-settings`) and the sound.
  **Its foot is Feedback and About** (`.settings-foot`), and it is always
  there: with nothing to set the place still holds About. The foot is
  `position: sticky`: the switches are longer than a phone's sheet, and a foot
  that scrolled with them was under the fold.
- **A sheet sits above the tab bar** and is as tall as what it holds, up to the
  room under the status bar; the chat's is that tall always, for its log to
  scroll. It covers the dock: a sheet is for reading, the dock for playing.
- **The desktop menu is as tall as what it holds**, up to the screen (less the
  dock under 1444px, where the two would meet): a short place makes a short
  panel, and only the board, or the account, scrolls inside it.
- **Escape closes the innermost** (`useEscape` keeps a stack): a sub-panel such
  as the country picker steps back, then the sheet closes. A modal dialog
  still takes Escape first.
- **Every `Modal` is drawn in a portal on the body**, so a dialog opened from
  inside the menu or a sheet covers the page and not its panel.
- **The other player's card** is the same `Modal`; on a phone its CSS makes it a
  bottom sheet as tall as the card.
- **The cards beside the pointer are the one exception**: the card on who holds a
  tile, and in orbit the card on a territory, open beside the pointer, not in a zone.
  See [Who holds a tile](tile-holder.md) and [A territory, from orbit](territory-card.md).
