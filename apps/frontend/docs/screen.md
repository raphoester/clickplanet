# The screen: four zones

**Nothing is drawn over the globe but four zones**, and a new feature goes in one
of them rather than in a fifth panel. The rule is in [`DESIGN.md`](../DESIGN.md);
this is where each zone lives. `PlayScreen` composes them, and `useCompact`
(`compact.ts`, the `COMPACT` query, kept live) picks the phone or the desktop form.

**The phone form is for a screen 768px wide or less, or 600px high or less.**
CrazyGames framed the game at 16:9, as small as 821×462, and a phone can be held
sideways at 800×450. Below 600px high the desktop menu and chat cover most of the
globe and the season chip covers the menu's title. At 606px high (CrazyGames'
1077×606) the desktop form still shows the board's leader, a row under it, the
chat and the globe between them, so it keeps that form. Every stylesheet's phone
rules use the same `@media` query as `COMPACT`; `compact.test.ts` fails on one
that does not.

**Held sideways, the phone form turns its bars into a rail** (`LANDSCAPE` in `compact.ts`, 600px high or less, wider
than high, which only a compact screen is, and wider than 480px). The width floor is for an upright phone with its
keyboard open: a browser that shrinks the viewport above the keyboard (Brave on an iPhone 17: 402×375) makes it wider
than high, and without the floor the chat turned into the rail's drawer while the player typed (2026-11-20). An
upright phone is 480px wide or less (`NARROW_PHONE`); a phone on its side is 568px wide or more. A phone on its side is 320 to 430px high: the status bar, the dock and the
tab bar stacked as upright left the globe a strip 60 to 100px high between them (iPhone SE 568×320, iPhone 14 750×340,
Pixel 7 863×360, CrazyGames' 821×462; 2026-11-19). The same components, so a turn keeps the open sheet, laid out
by CSS alone:

- **The tabs are a rail down the left edge** (`--rail-width`, 64px, and `--rail-room` with the notch's inset), the six
  icons centred down it.
- **The status bar is two pills over the globe**, the logo, flag, country and rank (`.status-bar-lead`) on the left
  and the season's chip on the right, and no bar between them: `--status-bottom` is under the pills.
- **The dock is as short as its slots**: the clicks and their line ("+1 in 1s") on one row, the level's line under
  them, the slots on the right of both, 66px high, 75px with the level. It is centred right of the rail, at most 560px
  wide (`--dock-left`, `--landscape-dock-width`), and the chat's balloons start at its left edge.
- **A sheet is a drawer out of the rail**, as tall as the screen and at most 400px wide, so the globe stays in view
  beside it; it covers the left pill, and the dock hides while it is open, as a sheet covers it upright.
- **What is centred at the top keeps clear of the rail** as of the desktop menu: `--menu-right` is the rail's edge.
  The quiz is up to 520px wide and tighter, so a three-line question ends above a 320px screen's bottom.
- **The player's card rises in the middle**, 520px wide, not as a strip across the screen.

Which form a turn lands on is unchanged: a phone is compact upright and sideways, so `useCompact` does not change and
nothing closes. Checked with Playwright's devices on their sides (iPhone SE, 14 and 15 Pro Max, Pixel 7, Galaxy S9+),
882×344, 821×462, 700×500 and 1024×600; upright and the desktop measure as before, box for box.

| Zone | Phone | Desktop |
|---|---|---|
| **Status** | `hud/StatusBar`: logo, flag, country and rank (opens the board), the shop's bag while it sells (opens the shop's modal), `SeasonChip` at its end (`BattleChip` while a battle plays); sideways, two pills, the bag in the left one | `Menu`'s "playing for" (a pill beside the rail while the panel is folded), `SeasonChip` at the top, centred in the room right of the menu (`BattleChip` while a battle plays) |
| **Moments** | under the status bar (`--status-bottom`) | under the season chip |
| **Play** | the dock (`ClickBudgetMeter` + `Inventory`, and the player's level and its bar across the dock under them) above the tab bar, the chat's peek above it | the dock at the bottom centre, the same |
| **Places** | `hud/TabBar` (Board, Missions, Chat, Sign in / You, Share, Settings: icons alone, of one width, each named by its label), each a `hud/Sheet`; sideways, a rail on the left and each sheet a drawer beside it | `Menu`'s rail (logo, Board, Missions, Share or Invite, Settings, then Shop while it sells, which opens the shop's modal, and Sign in / You at its foot) on the left with the open place beside it, the chat on the right |

- **What is open is one store**, `useScreen` (`hud/screenStore.ts`, see
  [Stores](stores.md)): the phone's sheet (`sheet`), the desktop menu's place
  (`menuTab`), the shop's modal (`shopOpen`, see [The shop](shop.md#the-shops-modal)), the popovers (`popovers`: the desktop's season, battle and
  "Your clicks", and on both forms the tray of every bonus, which All opens over the dock; see
  [Charges](charges.md#the-inventory)), and the player's last open or fold of the desktop menu's panel and of the chat (`menuOpen`,
  `chatOpen`, unset until the player chooses). The tab bar, the status bar, the chips, the dock, the menu and the chat
  read and change it themselves; `PlayScreen` reads the sheet to draw it. A change
  between the phone and the desktop form closes the sheet and the popovers. "Your clicks" and the bonuses' tray open one
  at a time, and a sheet that opens closes the tray: the sheet covers the dock.
- **No HUD piece covers another, but a popover while it is open.** "Your clicks" and the bonuses' tray float over the
  dock and cover what is above it until they close; the dock keeps its height for them. What stacks on the dock follows
  its drawn height, `--dock-drawn-height` (`useEdge` on the dock), which the level's line grows: the chat's balloons
  over a phone's dock, the chat between 769 and 1099px wide (`--chat-lift`), and the desktop menu's room
  (`--menu-room`), which shrinks and scrolls.
- **One sheet at a time on a phone**: the five
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
- **A sheet sits above the tab bar** and is as tall as what it holds, up to the
  room under the status bar; the chat's is that tall always, for its log to
  scroll. It covers the dock: a sheet is for reading, the dock for playing.
- **The desktop menu is a rail and a panel that folds.** The rail is the
  places as a vertical `tablist`; the panel beside it shows the open one,
  headed by its title (the board's Countries / Players heading on the board,
  with the Hall of Fame's crown), a hide button, and
  "playing for". The hide button, or the open place pressed again, folds the
  panel away: the rail stays, with the flag, its rank and Change as a pill
  beside it, and any place opens the panel again. Folded or open is kept in
  `localStorage` (`clickplanet-menu-open`) from one visit to the next, and the chat's fold the same way
  (`clickplanet-chat-open`).
- **In a small frame the board and the chat start folded**, until the player opens or folds
  one: then that choice wins at every later visit, in any frame. The globe is the screen, and both are one press away
  (the rail stays; the chat folds to its header). The frame is small under 1600px wide (`SMALL_FRAME`, `useSmallFrame`
  in `compact.ts`, followed live, so a frame made full screen opens them). The open board and chat take about 900 to
  960px side by side; under 1600px they leave the globe less than 40% of the width. Measured on the fake (2026-11-13),
  the strip between them is 226px at 1077×606 and 270px at 1216×684 (CrazyGames' frames outside full screen), 380px
  at 1280×720, 425px at 1366×768 and 576px at 1536×864, and 960px, half the width, at 1920×1080. On CrazyGames'
  1077×606 a reviewer saw a strip of globe 230px wide between the board and the chat, and the board's tooltip as the
  first text (2026-11-12). The phone form keeps its sheets closed anyway.
- **What is centred at the top centres on the globe and the dock, clear of the desktop menu.** The menu
  publishes its right edge, open or folded (`MENU_RIGHT`, `--menu-right`, through `useEdge`), and `index.css`
  turns it into `--top-centre` and `--top-room`, which place the season and battle chips, the quiz, the mission
  progress line, the bomb news, "under attack", "connection lost" and the battle's end; the news lines go under the
  quiz's bottom edge (see [Quizzes](quizzes.md)). "Under attack" once spanned the screen and covered the open board
  (2026-11-13). The centre is the screen's, unless that
  leaves less than 240px (half the 440px quiz and its margins) right of the menu: then it is 240px right of the
  menu, and never past the middle of the room right of it. With no menu, on a phone, it is the screen's centre.
  The menu is `clamp(340px, 40vw, 540px)` wide so the board shows the countries' names beside `earth-2`'s six-digit
  counts (see
  [Season standings](season-standings.md)); a chip centred on the screen, 368px wide, left it at most 417px at
  1440px wide (2026-11-11). Centred in the whole room right of the menu, the chips sat right of the globe on a
  wide screen. The 40vw always leaves the chip its room right of the menu from 834px wide.
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
