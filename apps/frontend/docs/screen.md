# The screen: four zones

**Nothing is drawn over the globe but four zones**, and a new feature goes in one
of them rather than in a fifth panel. The rule is in [`DESIGN.md`](../DESIGN.md);
this is where each zone lives. `Viewer` composes them, and `useCompact`
(`compact.ts`, the `COMPACT` query, kept live) picks the phone or the desktop form.

**The phone form is for a screen 768px wide or less, or 600px high or less.**
CrazyGames frames the game at 16:9, as small as 821×462, and a phone can be held
sideways at 800×450. Below 600px high the desktop menu and chat cover most of the
globe and the season chip covers the menu's title. At 606px high (CrazyGames'
1077×606) the desktop form still shows the board's leader, a row under it, the
chat and the globe between them, so it keeps that form. Every stylesheet's phone
rules use the same `@media` query as `COMPACT`; `compact.test.ts` fails on one
that does not.

| Zone | Phone | Desktop |
|---|---|---|
| **Status** | `hud/StatusBar`: logo, flag, country and rank (opens the board), `SeasonChip` at its end (`BattleChip` while a battle plays) | `Menu`'s "playing for" (a pill beside the rail while the panel is folded), `SeasonChip` at the top centre (`BattleChip` while a battle plays) |
| **Moments** | under the status bar (`--status-bottom`) | under the season chip |
| **Play** | the dock (`ClickBudgetMeter` + `Inventory`) above the tab bar, the chat's peek above it | the dock at the bottom centre |
| **Places** | `hud/TabBar` (Board, Missions, Chat, Sign in / You, Share, Settings), each a `hud/Sheet` | `Menu`'s rail (logo, Board, Missions, Share or Invite, Settings, and Sign in / You at its foot) on the left with the open place beside it, the chat on the right |

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
  and About at the foot of Settings, Discord, TikTok, YouTube and Instagram in
  the chat's bar and in About. A new entry goes to the place of its kind.
- **Settings is every switch the player keeps**: the display (the store
  `useDisplaySettings`, `domain/displaySettings.ts`, in `clickplanet-display-settings`, see
  [Stores](stores.md)) and the sound.
  **Its foot is Feedback and About** (`.settings-foot`), and it is always
  there. The foot is
  `position: sticky`: the switches are longer than a phone's sheet, and a foot
  that scrolled with them was under the fold.
- **What stands on the dock's top edge is its crown** (`--dock-crown`, 0 unless a mood puts props there, see
  [Halloween](halloween.md)): the menu, the chat above the dock and the phone's chat toast keep clear of it.
- **A sheet sits above the tab bar** and is as tall as what it holds, up to the
  room under the status bar; the chat's is that tall always, for its log to
  scroll. It covers the dock: a sheet is for reading, the dock for playing.
- **The desktop menu is a rail and a panel that folds.** The rail is the
  places as a vertical `tablist`; the panel beside it shows the open one,
  headed by its title (the board's Countries / Players heading on the board,
  with the Hall of Fame's crown and the order switch), a hide button, and
  "playing for". The hide button, or the open place pressed again, folds the
  panel away: the rail stays, with the flag, its rank and Change as a pill
  beside it, and any place opens the panel again. Folded or open is kept in
  `localStorage` (`clickplanet-menu-open`) from one visit to the next.
- **You is not a place on a desktop.** It sits at the rail's foot and opens
  `account/ProfileModal`, the player's own card at up to 1200px wide (the
  screen less 64px), read at a glance:
  - **the hero**: the worn medal at 112px, its name and rank, the flag played
    for, and four stats (tiles taken, streak and best streak with the flame,
    playing since);
  - **the medals earned**, at 76px: a press wears one (`WearTitle`), the worn
    one ringed in gold;
  - **the tracks** (`ProfileTracks`), a list and a detail side by side: on the
    left one tab per track (the current rank's medal, its name, a bar to the next
    rank), on the right every rank of the track picked, top to bottom, a bar down
    the medals filled up to the progress, and what is left in its header. The
    first track is picked when the profile opens. The detail stays in view while
    the list scrolls. It replaced cards in a grid whose "All ranks" opened the
    path under the last row, far from the card pressed once a player had 8
    tracks;
  - **the fronts**, side by side: the first three countries fought for and
    against, "See all" for the rest;
  - **the account** in a 340px column beside them (`ProfileAccount`): the
    username, the color, the providers as chips ("+ Discord" links one, "+ Email"
    opens its form), the clips switch, and sign out, sign out everywhere and
    delete as small buttons at its foot.

  Under 1100px the hero wraps and the account goes under the card. A guest gets
  the sign-in alone, at a modal's width. The phone keeps its You sheet
  (`YouPlace`: Progress and Account tabs, with folds, which a narrow sheet needs).
  Both read the titles and fronts through `account/useProgress.ts`.
- **The desktop menu is as tall as what it holds**, up to the screen less the
  dock and its crown, at every width: a short place makes a short panel, and
  only the board, or the place, scrolls inside it. It once kept clear of the
  dock only under 1444px, and ran under it on a short window wider than that.
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
